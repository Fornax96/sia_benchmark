package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"github.com/Fornaxian/config"
	sia "gitlab.com/NebulousLabs/Sia/node/api/client"
)

type Configuration struct {
	SiaAPIURL            string `toml:"sia_api_url"`
	SiaAPIPassword       string `toml:"sia_api_password"`
	SiaAPIUserAgent      string `toml:"sia_api_user_agent"`
	StopSiaOnFailure     bool   `toml:"stop_sia_on_failure"`
	Allowance            int    `toml:"allowance"`
	AllowancePeriod      int    `toml:"allowance_period"`
	HostCount            int    `toml:"host_count"`
	FileDataPieces       int    `toml:"file_data_pieces"`
	FileParityPieces     int    `toml:"file_parity_pieces"`
	FileSize             uint64 `toml:"file_size"`
	MaxConcurrentUploads int    `toml:"max_concurrent_uploads"`
	MinUploadRate        uint64 `toml:"min_upload_rate"`
	MeasurementInterval  uint   `toml:"measurement_interval"`
}

const defaultConfig = `# Sia benchmark tool configuration
sia_api_url            = "127.0.0.1:9980"
sia_api_password       = ""
sia_api_user_agent     = "Sia-Agent"
stop_sia_on_failure    = true # Whether to stop the Sia daemon if the test fails
allowance              = 1000 # SC
allowance_period       = 12096 # In blocks, this is three months
host_count             = 50
file_data_pieces       = 10
file_parity_pieces     = 20
file_size              = 1000000000 # This is 1 GB
max_concurrent_uploads = 10
min_upload_rate        = 1000000 # 1 MB per second
measurement_interval   = 60 # seconds
`

func main() {
	// Load the configuration
	var conf = Configuration{}
	_, err := config.New(defaultConfig, "", "benchmark.toml", &conf, true)
	if err != nil {
		panic(err)
	}

	var interval = time.Duration(conf.MeasurementInterval) * time.Second

	sc := sia.New(conf.SiaAPIURL)
	sc.Password = conf.SiaAPIPassword
	sc.UserAgent = conf.SiaAPIUserAgent

	version, err := sc.DaemonVersionGet()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Connected to Sia %s (rev %s)\n", version.Version, version.GitRevision)

	// Open the metrics CSV
	created := false
	f, err := os.OpenFile("metrics.csv", os.O_WRONLY, os.ModeAppend)
	if os.IsNotExist(err) {
		f, err = os.Create("metrics.csv")
		if err != nil {
			panic(err)
		}
		created = true
	} else if err != nil {
		panic(err)
	}

	// Append the data to the end of the file
	f.Seek(0, os.SEEK_END)

	csvWriter := csv.NewWriter(f)

	if created {
		// New file, print headers
		err = csvWriter.Write(MetricsHeaders())
		if err != nil {
			panic(err)
		}
		csvWriter.Flush()
	}

	// In this loop we collect stats on the
	//  - Files
	//  - Contracts
	//  - Wallet
	//  - Allowance
	//
	// We store all this in a Metrics struct. When the information is complete
	// we write the metrics to the CSV
	var lastSize uint64
	var bwLog = make([]uint64, 3600/conf.MeasurementInterval)
	var bwLogIndex = -1
	var bwAverage uint64
	var bwFirstCycle = true
	for {
		// Sleep until the next full minute
		time.Sleep(time.Until(time.Now().Add(interval).Truncate(interval)))

		metrics, err := collectMetrics(sc)
		if err != nil {
			fmt.Printf("Error while collecting metrics: %s\n", err)
			continue
		}

		err = metrics.WriteCSV(csvWriter)
		if err != nil {
			fmt.Printf("Error while writing to CSV: %s\n", err)
		}
		if err = csvWriter.Error(); err != nil {
			fmt.Printf("Error while flushing CSV: %s\n", err)
		}

		bwLogIndex++
		if bwLogIndex == len(bwLog) {
			bwLogIndex = 0
			bwFirstCycle = false
		}

		if lastSize != 0 {
			bwLog[bwLogIndex] = (metrics.ContractTotalSize - lastSize) / 60
		}

		// Calculate average bandwidth
		bwAverage = 0
		for _, bw := range bwLog {
			bwAverage += bw
		}
		if bwFirstCycle && bwLogIndex != 0 {
			bwAverage = bwAverage / uint64(bwLogIndex)
		} else {
			bwAverage = bwAverage / uint64(len(bwLog))
		}

		fmt.Printf("[%s] Latency: %s Files: %d, Uploading: %d, Speed current: %s/s, Average: %s/s\n",
			metrics.Timestamp.Format("2006-01-02 15:04:05 -0700 MST"),
			metrics.APILatency,
			metrics.FileCount,
			metrics.FileUploadsInProgressCount,
			formatData(bwLog[bwLogIndex]),
			formatData(bwAverage),
		)

		// Exit the test if bandwidth falls below the configured threshold
		if !bwFirstCycle && bwAverage < conf.MinUploadRate {
			fmt.Printf(
				"Average upload speed of %s/s fell below configured threshold of %s/s\n",
				formatData(bwAverage), formatData(conf.MinUploadRate))
			fmt.Printf(
				"The test has ended with a total of %s uploaded in file data and %s uploaded in contract data\n",
				formatData(metrics.FileTotalBytes), formatData(metrics.ContractTotalSize))

			if conf.StopSiaOnFailure {
				fmt.Println("Shutting down Sia...")
				err = sc.DaemonStopGet()
				if err != nil {
					fmt.Printf("Error stopping Sia daemon: %s\n", err)
				}
			}
			os.Exit(0)
		}

		lastSize = metrics.ContractTotalSize
	}
}

func collectMetrics(sc *sia.Client) (metrics Metrics, err error) {
	metrics.Timestamp = time.Now()

	// Collect file stats
	files, err := sc.RenterFilesGet(true)
	if err != nil {
		return metrics, err
	}
	for _, file := range files.Files {
		metrics.FileCount++
		metrics.FileTotalBytes += file.Filesize
		metrics.FileUploadedBytes += file.UploadedBytes
		if file.UploadProgress < 100 {
			metrics.FileUploadsInProgressCount++
		}
	}

	// Collect contract stats
	contracts, err := sc.RenterContractsGet()
	if err != nil {
		return metrics, err
	}
	metrics.ContractCountActive = len(contracts.ActiveContracts)
	metrics.ContractCountInactive = len(contracts.InactiveContracts)

	for _, contract := range append(contracts.ActiveContracts, contracts.InactiveContracts...) {
		metrics.ContractTotalSize += contract.Size
		metrics.ContractFeeSpending = metrics.ContractFeeSpending.Add(contract.Fees)
		metrics.ContractStorageSpending = metrics.ContractStorageSpending.Add(contract.StorageSpending)
		metrics.ContractUploadSpending = metrics.ContractUploadSpending.Add(contract.UploadSpending)
		metrics.ContractDownloadSpending = metrics.ContractDownloadSpending.Add(contract.DownloadSpending)
		metrics.ContractRemainingFunds = metrics.ContractRemainingFunds.Add(contract.RenterFunds)
	}

	// Add up the totals
	metrics.ContractTotalSpending = metrics.ContractTotalSpending.
		Add(metrics.ContractFeeSpending).
		Add(metrics.ContractStorageSpending).
		Add(metrics.ContractUploadSpending).
		Add(metrics.ContractDownloadSpending)

	// Collect wallet stats
	wallet, err := sc.WalletGet()
	if err != nil {
		return metrics, err
	}
	metrics.WalletSiacoinBalance = wallet.ConfirmedSiacoinBalance
	metrics.WalletOutgoingSiacoins = wallet.UnconfirmedOutgoingSiacoins
	metrics.WalletIncomingSiacoins = wallet.UnconfirmedIncomingSiacoins

	// Collect renter stats
	renter, err := sc.RenterGet()
	if err != nil {
		return metrics, err
	}
	metrics.RenterAllowance = renter.Settings.Allowance.Funds
	metrics.RenterContractFees = renter.FinancialMetrics.ContractFees
	metrics.RenterTotalAllocated = renter.FinancialMetrics.TotalAllocated
	metrics.RenterDownloadSpending = renter.FinancialMetrics.DownloadSpending
	metrics.RenterStorageSpending = renter.FinancialMetrics.StorageSpending
	metrics.RenterUploadSpending = renter.FinancialMetrics.UploadSpending
	metrics.RenterUnspent = renter.FinancialMetrics.Unspent

	metrics.APILatency = time.Since(metrics.Timestamp)

	return metrics, nil
}

// FormatData converts a raw amount of bytes to an easily readable string
func formatData(v uint64) string {
	var fmtSize = func(n float64, u string) string {
		var f string
		if n > 100 {
			f = "%5.1f"
		} else if n > 10 {
			f = "%5.2f"
		} else {
			f = "%5.3f"
		}
		return fmt.Sprintf(f+" "+u, n)
	}
	if v >= 1e12 {
		return fmtSize(float64(v)/1e12, "TB")
	} else if v >= 1e9 {
		return fmtSize(float64(v)/1e9, "GB")
	} else if v >= 1e6 {
		return fmtSize(float64(v)/1e6, "MB")
	} else if v >= 1e3 {
		return fmtSize(float64(v)/1e3, "kB")
	}
	return fmt.Sprintf("%5d  B", v)
}
