package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"

	sia "gitlab.com/NebulousLabs/Sia/node/api/client"
)

func main() {

	sc := sia.New("127.0.0.1:9980")
	sc.Password = "d0290de6ff4731a4de8da2f68cb7c3fe"

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
	for {
		// Sleep until the next full minute
		time.Sleep(time.Until(time.Now().Add(time.Minute).Truncate(time.Minute)))

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

		fmt.Printf("[%s] Latency: %s Files: %d, Uploading: %d, Speed: %s/s\n",
			metrics.Timestamp.Format("2006-01-02 15:04:05 -0700 MST"),
			metrics.APILatency,
			metrics.FileCount,
			metrics.FileUploadsInProgressCount,
			formatData((metrics.FileUploadedBytes-lastSize)/60),
		)
		lastSize = metrics.FileUploadedBytes

		fmt.Println(metrics)
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
	if v > 1e12 {
		return fmtSize(float64(v)/1e12, "TB")
	} else if v > 1e9 {
		return fmtSize(float64(v)/1e9, "GB")
	} else if v > 1e6 {
		return fmtSize(float64(v)/1e6, "MB")
	} else if v > 1e3 {
		return fmtSize(float64(v)/1e3, "kB")
	}
	return fmt.Sprintf("%5d  B", v)
}
