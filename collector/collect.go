package collector

import (
	"time"

	"gitlab.com/NebulousLabs/Sia/node/api"
	sia "gitlab.com/NebulousLabs/Sia/node/api/client"
)

// CollectMetrics collects stats on the Files, Contracts, Wallet and Allowance
// of the Sia node. It stores a summary of all the information in the Metrics
// struct and returns it
func CollectMetrics(sc *sia.Client) (metrics Metrics, err error) {
	metrics.Timestamp = time.Now()

	// Collect file stats
	files, err := sc.RenterFilesGet(true)
	if err != nil {
		return metrics, err
	}
	for _, file := range files.Files {
		metrics.FileTotalBytes += uint64(float64(file.Filesize) * (file.UploadProgress / 100))
		metrics.FileCount++
		metrics.FileUploadedBytes += file.UploadedBytes
		if file.UploadProgress < 100 {
			metrics.FileUploadsInProgressCount++
		}
	}

	// Collect contract stats
	contracts, err := sc.RenterAllContractsGet()
	if err != nil {
		return metrics, err
	}

	var addTotals = func(contract api.RenterContract, countSize bool) {
		if countSize {
			metrics.ContractSizeTotal += contract.Size
		}
		metrics.ContractCountTotal++
		metrics.ContractFundsRemainingTotal = metrics.ContractFundsRemainingTotal.Add(contract.RenterFunds)
		metrics.ContractStorageSpendingTotal = metrics.ContractStorageSpendingTotal.Add(contract.StorageSpending)
		metrics.ContractFeeSpendingTotal = metrics.ContractFeeSpendingTotal.Add(contract.Fees)
		metrics.ContractUploadSpendingTotal = metrics.ContractUploadSpendingTotal.Add(contract.UploadSpending)
		metrics.ContractDownloadSpendingTotal = metrics.ContractDownloadSpendingTotal.Add(contract.DownloadSpending)
	}

	// Tally up all the contract stats
	for _, contract := range contracts.ActiveContracts {
		addTotals(contract, true)
		metrics.ContractCountActive++
		metrics.ContractSizeActive += contract.Size
		metrics.ContractFundsRemainingActive = metrics.ContractFundsRemainingActive.Add(contract.RenterFunds)
		metrics.ContractStorageSpendingActive = metrics.ContractStorageSpendingActive.Add(contract.StorageSpending)
		metrics.ContractFeeSpendingActive = metrics.ContractFeeSpendingActive.Add(contract.Fees)
		metrics.ContractUploadSpendingActive = metrics.ContractUploadSpendingActive.Add(contract.UploadSpending)
		metrics.ContractDownloadSpendingActive = metrics.ContractDownloadSpendingActive.Add(contract.DownloadSpending)
	}
	for _, contract := range contracts.PassiveContracts {
		addTotals(contract, true)
		metrics.ContractCountPassive++
		metrics.ContractSizePassive += contract.Size
		metrics.ContractFundsRemainingPassive = metrics.ContractFundsRemainingPassive.Add(contract.RenterFunds)
		metrics.ContractStorageSpendingPassive = metrics.ContractStorageSpendingPassive.Add(contract.StorageSpending)
		metrics.ContractFeeSpendingPassive = metrics.ContractFeeSpendingPassive.Add(contract.Fees)
		metrics.ContractUploadSpendingPassive = metrics.ContractUploadSpendingPassive.Add(contract.UploadSpending)
		metrics.ContractDownloadSpendingPassive = metrics.ContractDownloadSpendingPassive.Add(contract.DownloadSpending)
	}
	for _, contract := range contracts.RefreshedContracts {
		addTotals(contract, false)
		metrics.ContractCountRefreshed++
		metrics.ContractSizeRefreshed += contract.Size
		metrics.ContractFundsRemainingRefreshed = metrics.ContractFundsRemainingRefreshed.Add(contract.RenterFunds)
		metrics.ContractStorageSpendingRefreshed = metrics.ContractStorageSpendingRefreshed.Add(contract.StorageSpending)
		metrics.ContractFeeSpendingRefreshed = metrics.ContractFeeSpendingRefreshed.Add(contract.Fees)
		metrics.ContractUploadSpendingRefreshed = metrics.ContractUploadSpendingRefreshed.Add(contract.UploadSpending)
		metrics.ContractDownloadSpendingRefreshed = metrics.ContractDownloadSpendingRefreshed.Add(contract.DownloadSpending)
	}
	for _, contract := range contracts.DisabledContracts {
		addTotals(contract, true)
		metrics.ContractCountDisabled++
		metrics.ContractSizeDisabled += contract.Size
		metrics.ContractFundsRemainingDisabled = metrics.ContractFundsRemainingDisabled.Add(contract.RenterFunds)
		metrics.ContractStorageSpendingDisabled = metrics.ContractStorageSpendingDisabled.Add(contract.StorageSpending)
		metrics.ContractFeeSpendingDisabled = metrics.ContractFeeSpendingDisabled.Add(contract.Fees)
		metrics.ContractUploadSpendingDisabled = metrics.ContractUploadSpendingDisabled.Add(contract.UploadSpending)
		metrics.ContractDownloadSpendingDisabled = metrics.ContractDownloadSpendingDisabled.Add(contract.DownloadSpending)
	}
	for _, contract := range contracts.ExpiredContracts {
		addTotals(contract, false)
		metrics.ContractCountExpired++
		metrics.ContractSizeExpired += contract.Size
		metrics.ContractFundsRemainingExpired = metrics.ContractFundsRemainingExpired.Add(contract.RenterFunds)
		metrics.ContractStorageSpendingExpired = metrics.ContractStorageSpendingExpired.Add(contract.StorageSpending)
		metrics.ContractFeeSpendingExpired = metrics.ContractFeeSpendingExpired.Add(contract.Fees)
		metrics.ContractUploadSpendingExpired = metrics.ContractUploadSpendingExpired.Add(contract.UploadSpending)
		metrics.ContractDownloadSpendingExpired = metrics.ContractDownloadSpendingExpired.Add(contract.DownloadSpending)
	}
	for _, contract := range contracts.ExpiredRefreshedContracts {
		addTotals(contract, false)
		metrics.ContractCountExpiredRefreshed++
		metrics.ContractSizeExpiredRefreshed += contract.Size
		metrics.ContractFundsRemainingExpiredRefreshed = metrics.ContractFundsRemainingExpiredRefreshed.Add(contract.RenterFunds)
		metrics.ContractStorageSpendingExpiredRefreshed = metrics.ContractStorageSpendingExpiredRefreshed.Add(contract.StorageSpending)
		metrics.ContractFeeSpendingExpiredRefreshed = metrics.ContractFeeSpendingExpiredRefreshed.Add(contract.Fees)
		metrics.ContractUploadSpendingExpiredRefreshed = metrics.ContractUploadSpendingExpiredRefreshed.Add(contract.UploadSpending)
		metrics.ContractDownloadSpendingExpiredRefreshed = metrics.ContractDownloadSpendingExpiredRefreshed.Add(contract.DownloadSpending)
	}

	// Add up the grand total
	metrics.ContractSpendingTotal = metrics.ContractStorageSpendingTotal.
		Add(metrics.ContractFeeSpendingTotal).
		Add(metrics.ContractUploadSpendingTotal).
		Add(metrics.ContractDownloadSpendingTotal)

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
