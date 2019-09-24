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

	ContractCountTotal            int `csv:"contract_count_total"`
	ContractCountActive           int `csv:"contract_count_active"`
	ContractCountPassive          int `csv:"contract_count_passive"`
	ContractCountRefreshed        int `csv:"contract_count_refreshed"`
	ContractCountDisabled         int `csv:"contract_count_disabled"`
	ContractCountExpired          int `csv:"contract_count_expired"`
	ContractCountExpiredRefreshed int `csv:"contract_count_expired_refreshed"`

	ContractSizeTotal            uint64 `csv:"contract_size_total"`
	ContractSizeActive           uint64 `csv:"contract_size_active"`
	ContractSizePassive          uint64 `csv:"contract_size_passive"`
	ContractSizeRefreshed        uint64 `csv:"contract_size_refreshed"`
	ContractSizeDisabled         uint64 `csv:"contract_size_disabled"`
	ContractSizeExpired          uint64 `csv:"contract_size_expired"`
	ContractSizeExpiredRefreshed uint64 `csv:"contract_size_expired_refreshed"`

	ContractFundsRemainingTotal            types.Currency `csv:"contract_funds_remaining_total"`
	ContractFundsRemainingActive           types.Currency `csv:"contract_funds_remaining_active"`
	ContractFundsRemainingPassive          types.Currency `csv:"contract_funds_remaining_passive"`
	ContractFundsRemainingRefreshed        types.Currency `csv:"contract_funds_remaining_refreshed"`
	ContractFundsRemainingDisabled         types.Currency `csv:"contract_funds_remaining_disabled"`
	ContractFundsRemainingExpired          types.Currency `csv:"contract_funds_remaining_expired"`
	ContractFundsRemainingExpiredRefreshed types.Currency `csv:"contract_funds_remaining_expired_refreshed"`

	ContractSpendingTotal types.Currency `csv:"contract_spending_total"`

	ContractStorageSpendingTotal            types.Currency `csv:"contract_storage_spending_total"`
	ContractStorageSpendingActive           types.Currency `csv:"contract_storage_spending_active"`
	ContractStorageSpendingPassive          types.Currency `csv:"contract_storage_spending_passive"`
	ContractStorageSpendingRefreshed        types.Currency `csv:"contract_storage_spending_refreshed"`
	ContractStorageSpendingDisabled         types.Currency `csv:"contract_storage_spending_disabled"`
	ContractStorageSpendingExpired          types.Currency `csv:"contract_storage_spending_expired"`
	ContractStorageSpendingExpiredRefreshed types.Currency `csv:"contract_storage_spending_expired_refreshed"`

	ContractFeeSpendingTotal            types.Currency `csv:"contract_fee_spending_total"`
	ContractFeeSpendingActive           types.Currency `csv:"contract_fee_spending_active"`
	ContractFeeSpendingPassive          types.Currency `csv:"contract_fee_spending_passive"`
	ContractFeeSpendingRefreshed        types.Currency `csv:"contract_fee_spending_refreshed"`
	ContractFeeSpendingDisabled         types.Currency `csv:"contract_fee_spending_disabled"`
	ContractFeeSpendingExpired          types.Currency `csv:"contract_fee_spending_expired"`
	ContractFeeSpendingExpiredRefreshed types.Currency `csv:"contract_fee_spending_expired_refreshed"`

	ContractUploadSpendingTotal            types.Currency `csv:"contract_upload_spending_total"`
	ContractUploadSpendingActive           types.Currency `csv:"contract_upload_spending_active"`
	ContractUploadSpendingPassive          types.Currency `csv:"contract_upload_spending_passive"`
	ContractUploadSpendingRefreshed        types.Currency `csv:"contract_upload_spending_refreshed"`
	ContractUploadSpendingDisabled         types.Currency `csv:"contract_upload_spending_disabled"`
	ContractUploadSpendingExpired          types.Currency `csv:"contract_upload_spending_expired"`
	ContractUploadSpendingExpiredRefreshed types.Currency `csv:"contract_upload_spending_expired_refreshed"`

	ContractDownloadSpendingTotal            types.Currency `csv:"contract_download_spending_total"`
	ContractDownloadSpendingActive           types.Currency `csv:"contract_download_spending_active"`
	ContractDownloadSpendingPassive          types.Currency `csv:"contract_download_spending_passive"`
	ContractDownloadSpendingRefreshed        types.Currency `csv:"contract_download_spending_refreshed"`
	ContractDownloadSpendingDisabled         types.Currency `csv:"contract_download_spending_disabled"`
	ContractDownloadSpendingExpired          types.Currency `csv:"contract_download_spending_expired"`
	ContractDownloadSpendingExpiredRefreshed types.Currency `csv:"contract_download_spending_expired_refreshed"`

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

		strconv.Itoa(m.ContractCountTotal),
		strconv.Itoa(m.ContractCountActive),
		strconv.Itoa(m.ContractCountPassive),
		strconv.Itoa(m.ContractCountRefreshed),
		strconv.Itoa(m.ContractCountDisabled),
		strconv.Itoa(m.ContractCountExpired),
		strconv.Itoa(m.ContractCountExpiredRefreshed),

		strconv.FormatUint(m.ContractSizeTotal, 10),
		strconv.FormatUint(m.ContractSizeActive, 10),
		strconv.FormatUint(m.ContractSizePassive, 10),
		strconv.FormatUint(m.ContractSizeRefreshed, 10),
		strconv.FormatUint(m.ContractSizeDisabled, 10),
		strconv.FormatUint(m.ContractSizeExpired, 10),
		strconv.FormatUint(m.ContractSizeExpiredRefreshed, 10),

		m.ContractFundsRemainingTotal.String(),
		m.ContractFundsRemainingActive.String(),
		m.ContractFundsRemainingPassive.String(),
		m.ContractFundsRemainingRefreshed.String(),
		m.ContractFundsRemainingDisabled.String(),
		m.ContractFundsRemainingExpired.String(),
		m.ContractFundsRemainingExpiredRefreshed.String(),

		m.ContractSpendingTotal.String(),

		m.ContractStorageSpendingTotal.String(),
		m.ContractStorageSpendingActive.String(),
		m.ContractStorageSpendingPassive.String(),
		m.ContractStorageSpendingRefreshed.String(),
		m.ContractStorageSpendingDisabled.String(),
		m.ContractStorageSpendingExpired.String(),
		m.ContractStorageSpendingExpiredRefreshed.String(),

		m.ContractFeeSpendingTotal.String(),
		m.ContractFeeSpendingActive.String(),
		m.ContractFeeSpendingPassive.String(),
		m.ContractFeeSpendingRefreshed.String(),
		m.ContractFeeSpendingDisabled.String(),
		m.ContractFeeSpendingExpired.String(),
		m.ContractFeeSpendingExpiredRefreshed.String(),

		m.ContractUploadSpendingTotal.String(),
		m.ContractUploadSpendingActive.String(),
		m.ContractUploadSpendingPassive.String(),
		m.ContractUploadSpendingRefreshed.String(),
		m.ContractUploadSpendingDisabled.String(),
		m.ContractUploadSpendingExpired.String(),
		m.ContractUploadSpendingExpiredRefreshed.String(),

		m.ContractDownloadSpendingTotal.String(),
		m.ContractDownloadSpendingActive.String(),
		m.ContractDownloadSpendingPassive.String(),
		m.ContractDownloadSpendingRefreshed.String(),
		m.ContractDownloadSpendingDisabled.String(),
		m.ContractDownloadSpendingExpired.String(),
		m.ContractDownloadSpendingExpiredRefreshed.String(),

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
