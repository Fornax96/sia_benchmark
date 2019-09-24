package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"gitlab.com/NebulousLabs/fastrand"

	"github.com/Fornax96/sia_benchmark/collector"
	"github.com/Fornaxian/config"
	"github.com/Fornaxian/log"
	sia "gitlab.com/NebulousLabs/Sia/node/api/client"
)

// Configuration for the benchmark
type Configuration struct {
	// Sia API config
	SiaAPIURL       string `toml:"sia_api_url"`
	SiaAPIPassword  string `toml:"sia_api_password"`
	SiaAPIUserAgent string `toml:"sia_api_user_agent"`

	WatchOnly bool `toml:"watch_only"`

	// Allowance settings
	Allowance        int    `toml:"allowance"`
	AllowancePeriod  int    `toml:"allowance_period"`
	HostCount        int    `toml:"host_count"`
	FileDataPieces   uint64 `toml:"file_data_pieces"`
	FileParityPieces uint64 `toml:"file_parity_pieces"`

	// Test parameters
	FileSize             uint64 `toml:"file_size"`
	MaxConcurrentUploads uint64 `toml:"max_concurrent_uploads"`
	MinUploadRate        uint64 `toml:"min_upload_rate"`
	MeasurementInterval  uint   `toml:"measurement_interval"`
	MeasurementPeriod    uint   `toml:"measurement_period"`

	// How many bytes the Sia node needs to upload before the test is
	// successful. If this is 0 the test will go on until the bandwidth
	// thtreshold is crossed
	SuccessSizeThreshold uint64 `toml:"success_size_threshold"`

	// Where the files will be generated and uploaded from
	FileUploadsDir string `toml:"file_uploads_dir"`

	// Exit condition
	StopSiaOnExit bool `toml:"stop_sia_on_exit"`

	LoggingVerbosity int `toml:"logging_verbosity"`
}

const defaultConfig = `# Sia benchmark tool configuration

# Sia API config
sia_api_url            = "127.0.0.1:9980"
sia_api_password       = ""
sia_api_user_agent     = "Sia-Agent"

# if watch_only is enabled the benchmark tool will not do any uploading. It will
# only monitor the Sia daemon. It will also never check the exit condition
watch_only             = false

# Allowance settings
allowance              = 1000 # SC
allowance_period       = 12096 # In blocks, this is three months
host_count             = 50
file_data_pieces       = 10
file_parity_pieces     = 20

# Test parameters
file_size              = 1000000000 # This is 1 GB
max_concurrent_uploads = 10
min_upload_rate        = 1000000 # 1 MB per second

# How often to poll the Sia API for new metrics
measurement_interval   = 60 # one minute

# Used for averaging the historic bandwidth numbers. If min upload rate is
# 1 MB/s and the measurement period is two hours the test will only end if
# bandwidth drops below 1 MB/s for two hours
measurement_period     = 7200 # two hours

# How many bytes the Sia node needs to upload before the test is successful. If
# this is 0 the test will go on until the bandwidth thtreshold is crossed
success_size_threshold = 1000000000000 # 1 TB

# Where the files will be generated and uploaded from
file_uploads_dir       = "upload_queue"

# Exit condition. Whether to stop the Sia daemon if the test ends
stop_sia_on_exit       = true

logging_verbosity      = 3 # 4 = debug, 3 = info, 2 = warning, 1 = error
`

func main() {
	// Load the configuration
	var conf = Configuration{}
	_, err := config.New(defaultConfig, "", "benchmark.toml", &conf, true)
	if err != nil {
		panic(err)
	}
	log.SetLogLevel(conf.LoggingVerbosity)

	// Check if uploads directory exists
	dir, err := os.Stat(conf.FileUploadsDir)
	if !conf.WatchOnly && err != nil {
		panic(err)
	}
	if !conf.WatchOnly && !dir.IsDir() {
		log.Error("Upload queue directory %s is not a directory", conf.FileUploadsDir)
		os.Exit(1)
	}

	conf.FileUploadsDir, err = filepath.Abs(conf.FileUploadsDir)
	if !conf.WatchOnly && err != nil {
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
	log.Info("Connected to Sia %s (rev %s)", version.Version, version.GitRevision)

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
		err = csvWriter.Write(collector.MetricsHeaders())
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

	// The bandwidth log saves bandwidth usage over the configured measurement
	// period. The numbers in this array are averaged every round and stored in
	// bwAverage to get the average bandwidth consumption. This number is used
	// for determining if the exit condition was reached.
	var bwLog = make([]uint64, conf.MeasurementPeriod/conf.MeasurementInterval)
	var bwLogIndex = -1
	var bwAverage uint64
	var bwFirstCycle = true
	var uploading = false
	for {
		// Sleep until the next full minute
		time.Sleep(time.Until(time.Now().Add(interval).Truncate(interval)))

		metrics, err := collector.CollectMetrics(sc)
		if err != nil {
			log.Warn("Error while collecting metrics: %s", err)
			continue
		}

		err = metrics.WriteCSV(csvWriter)
		if err != nil {
			panic(fmt.Errorf("error while writing to CSV: %s", err))
		}
		if err = csvWriter.Error(); err != nil {
			panic(fmt.Errorf("error while flushing CSV: %s", err))
		}

		// Reset the array index pointer to 0 when it's getting out of bounds
		bwLogIndex++
		if bwLogIndex == len(bwLog) {
			bwLogIndex = 0
			bwFirstCycle = false
		}

		// Overwrite the oldest digit in the bandwith log array
		if lastSize != 0 && lastSize <= metrics.ContractSizeTotal {
			bwLog[bwLogIndex] = (metrics.ContractSizeTotal - lastSize) / uint64(conf.MeasurementInterval)
		}
		lastSize = metrics.ContractSizeTotal

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

		// Print test statistics
		if bwLogIndex%30 == 0 {
			// Print headers every 30 rows
			log.Info("%-30s  %-14s  %-5s  %-9s  %-9s  %-13s  %-10s  %-13s  %-13s  %-10s  %-10s",
				"Timestamp",
				"Latency",
				"Files",
				"Uploading",
				"File Size",
				"Contract Size",
				"Efficiency",
				"Current Speed",
				"Average Speed",
				"Spent",
				"Unspent",
			)
		}
		if metrics.ContractSizeTotal == 0 {
			metrics.ContractSizeTotal = 1 // Avoid division by zero
		}
		log.Info("%-30s  %-14s  %5d  %9d  %9s  %13s  %9.2f%%  %11s/s  %11s/s  %10s  %10s",
			metrics.Timestamp.Format("2006-01-02 15:04:05 -0700 MST"),
			metrics.APILatency,
			metrics.FileCount,
			metrics.FileUploadsInProgressCount,
			formatData(metrics.FileTotalBytes),
			formatData(metrics.ContractSizeTotal),
			(float64(metrics.FileTotalBytes)/float64(metrics.ContractSizeTotal))*100,
			formatData(bwLog[bwLogIndex]),
			formatData(bwAverage),
			metrics.ContractSpendingTotal.HumanString(),
			metrics.ContractFundsRemainingTotal.HumanString(),
		)

		// This function exits the program if the exit conditions are met. The
		// test cannot end within one hour of starting
		if !bwFirstCycle && !conf.WatchOnly {
			testExitCondition(metrics, bwAverage, conf, sc)
		}

		// Clean up finished uploads
		if !conf.WatchOnly {
			err = collector.FinishUploads(sc, conf.FileUploadsDir)
			if err != nil {
				log.Error("Error while removing finished uploads: %s", err)
			}
		}

		// Test conditions not met, continue uploading files. Here files are
		// uploaded if:
		//  - There are not already files being uploaded
		//  - Watch Only mode is disabled
		//  - There are upload slots available
		//  - There are enough contracts to support the file
		//  - The total size of files is under the success threshold (to prevent
		//    overshooting). Or the size threshold is disabled
		if !uploading && !conf.WatchOnly &&
			metrics.FileUploadsInProgressCount < conf.MaxConcurrentUploads &&
			uint64(metrics.ContractCountActive) >= conf.FileDataPieces+conf.FileParityPieces &&
			(metrics.FileTotalBytes+(metrics.FileUploadsInProgressCount*conf.FileSize) < conf.SuccessSizeThreshold ||
				conf.SuccessSizeThreshold == 0) {
			uploading = true

			// This function can take a long time to run, so in order to not
			// hold up the metrics loop is runs in a separate thread
			go func() {
				// Upload files concurrently in order to utilize all available
				// CPU cores
				wg := sync.WaitGroup{}
				for i := uint64(0); i < conf.MaxConcurrentUploads-metrics.FileUploadsInProgressCount; i++ {
					wg.Add(1)
					go func() {
						err = collector.UploadFile(
							sc,
							conf.FileUploadsDir+"/"+strconv.Itoa(fastrand.Intn(999999999))+".dat",
							conf.FileDataPieces,
							conf.FileParityPieces,
							conf.FileSize,
						)
						if err != nil {
							log.Warn("Failed to upload file to Sia: %s", err)
						}
						wg.Done()
					}()
				}
				wg.Wait()
				uploading = false
			}()
		}
	}
}

func testExitCondition(
	metrics collector.Metrics,
	bwAverage uint64,
	conf Configuration,
	sc *sia.Client,
) {
	var err error

	// Exit the test if bandwidth falls below the configured threshold
	if bwAverage < conf.MinUploadRate {
		log.Warn(
			"Average upload speed of %s/s fell below configured threshold of %s/s",
			formatData(bwAverage), formatData(conf.MinUploadRate))
		log.Warn(
			"The test has ended with a total of %s uploaded in file data and %s uploaded in contract data",
			formatData(metrics.FileTotalBytes), formatData(metrics.ContractSizeTotal))

		if conf.StopSiaOnExit {
			log.Info("Shutting down Sia...")
			err = sc.DaemonStopGet()
			if err != nil {
				log.Error("Error stopping Sia daemon: %s", err)
			}
		}
		os.Exit(0)
	}

	// Exit the test if the total file size reaches the configured success
	// threshold
	if conf.SuccessSizeThreshold > 0 && metrics.FileTotalBytes >= conf.SuccessSizeThreshold {
		log.Info(
			"Total uploaded file size of %s met configured threshold of %s!",
			formatData(metrics.FileTotalBytes), formatData(conf.SuccessSizeThreshold))
		log.Info(
			"The test has ended with a total of %s uploaded in contract data and %s spent",
			formatData(metrics.ContractSizeTotal), metrics.ContractSpendingTotal.HumanString())

		if conf.StopSiaOnExit {
			log.Info("Shutting down Sia...")
			err = sc.DaemonStopGet()
			if err != nil {
				log.Error("Error stopping Sia daemon: %s", err)
			}
		}
		os.Exit(0)
	}
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
