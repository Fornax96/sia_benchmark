package collector

import (
	"encoding/csv"
	"reflect"
	"strconv"
	"time"

	"gitlab.com/NebulousLabs/Sia/types"
)

// Metrics contains all the metrics that will be collected when the metrics
// collector runs
type Metrics struct {
	Timestamp  time.Time     `csv:"timestamp"`
	APILatency time.Duration `csv:"api_latency"`

	FileCount                  uint64 `csv:"file_count"`
	FileTotalBytes             uint64 `csv:"file_total_bytes"`
	FileUploadsInProgressCount uint64 `csv:"file_uploads_in_progress_count"`
	FileUploadedBytes          uint64 `csv:"file_uploaded_bytes"`

	ContractCountActive      int            `csv:"contract_count_active"`
	ContractCountInactive    int            `csv:"contract_count_inactive"`
	ContractTotalSize        uint64         `csv:"contract_total_size"`
	ContractTotalSpending    types.Currency `csv:"contract_total_spending"`
	ContractFeeSpending      types.Currency `csv:"contract_fee_spending"`
	ContractStorageSpending  types.Currency `csv:"contract_storage_spending"`
	ContractUploadSpending   types.Currency `csv:"contract_upload_spending"`
	ContractDownloadSpending types.Currency `csv:"contract_download_spending"`
	ContractRemainingFunds   types.Currency `csv:"contract_remaining_funds"`

	WalletSiacoinBalance   types.Currency `csv:"wallet_siacoin_balance"`
	WalletOutgoingSiacoins types.Currency `csv:"wallet_outgoing_siacoins"`
	WalletIncomingSiacoins types.Currency `csv:"wallet_incoming_siacoins"`

	RenterAllowance        types.Currency `csv:"renter_allowance"`
	RenterContractFees     types.Currency `csv:"renter_contract_fees"`
	RenterTotalAllocated   types.Currency `csv:"renter_total_allocated"`
	RenterDownloadSpending types.Currency `csv:"renter_download_spending"`
	RenterStorageSpending  types.Currency `csv:"renter_storage_spending"`
	RenterUploadSpending   types.Currency `csv:"renter_upload_spending"`
	RenterUnspent          types.Currency `csv:"renter_unspent"`
}

// MetricsHeaders returns all the CSV headers of the Metrics struct so they can
// be written at the beginning of a new CSV file
func MetricsHeaders() (headers []string) {
	t := reflect.TypeOf(Metrics{})

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("csv")
		headers = append(headers, tag)
	}
	return headers
}

// Values marshals all the values to a string and returns them in an array so
// they can be written to the CSV
func (m Metrics) Values() (values []string) {
	return append(values,
		m.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
		m.APILatency.String(),
		strconv.FormatUint(m.FileCount, 10),
		strconv.FormatUint(m.FileTotalBytes, 10),
		strconv.FormatUint(m.FileUploadsInProgressCount, 10),
		strconv.FormatUint(m.FileUploadedBytes, 10),
		strconv.Itoa(m.ContractCountActive),
		strconv.Itoa(m.ContractCountInactive),
		strconv.FormatUint(m.ContractTotalSize, 10),
		m.ContractTotalSpending.String(),
		m.ContractFeeSpending.String(),
		m.ContractStorageSpending.String(),
		m.ContractUploadSpending.String(),
		m.ContractDownloadSpending.String(),
		m.ContractRemainingFunds.String(),
		m.WalletSiacoinBalance.String(),
		m.WalletOutgoingSiacoins.String(),
		m.WalletIncomingSiacoins.String(),
		m.RenterAllowance.String(),
		m.RenterContractFees.String(),
		m.RenterTotalAllocated.String(),
		m.RenterDownloadSpending.String(),
		m.RenterStorageSpending.String(),
		m.RenterUploadSpending.String(),
		m.RenterUnspent.String(),
	)
}

// WriteCSV adds a row to an existing CSV file with the stored values of Metrics
func (m Metrics) WriteCSV(f *csv.Writer) error {
	defer f.Flush()
	return f.Write(m.Values())
}
