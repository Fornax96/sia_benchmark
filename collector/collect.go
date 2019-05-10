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
		metrics.FileCount++
		metrics.FileUploadedBytes += file.UploadedBytes
		if file.UploadProgress < 100 {
			metrics.FileUploadsInProgressCount++
		} else {
			// Only include finished files because if we include unfinished
			// files the results will be skewed in sia's favour and the
			// efficiency numbers will be incorrect
			metrics.FileTotalBytes += file.Filesize
		}
	}

	// Collect contract stats
	contracts, err := sc.RenterInactiveContractsGet()
	if err != nil {
		return metrics, err
	}

	// Record active hosts
	activeHosts := make(map[string]struct{})
	for _, contract := range contracts.ActiveContracts {
		activeHosts[contract.HostPublicKey.String()] = struct{}{}
	}

	// Breakout renewed contracts and disabled Contracts
	var disabledContracts, renewedContracts []api.RenterContract
	for _, contract := range contracts.InactiveContracts {
		if _, ok := activeHosts[contract.HostPublicKey.String()]; ok {
			renewedContracts = append(renewedContracts, contract)
			continue
		}
		disabledContracts = append(disabledContracts, contract)
	}

	// Count contracts
	metrics.ContractCountActive = len(contracts.ActiveContracts)
	metrics.ContractCountRenewed = len(renewedContracts)
	metrics.ContractCountDisabled = len(disabledContracts)

	// Sum up spending
	for _, contract := range append(contracts.ActiveContracts, contracts.InactiveContracts...) {
		metrics.ContractFeeSpending = metrics.ContractFeeSpending.Add(contract.Fees)
		metrics.ContractStorageSpending = metrics.ContractStorageSpending.Add(contract.StorageSpending)
		metrics.ContractUploadSpending = metrics.ContractUploadSpending.Add(contract.UploadSpending)
		metrics.ContractDownloadSpending = metrics.ContractDownloadSpending.Add(contract.DownloadSpending)
		metrics.ContractRemainingFunds = metrics.ContractRemainingFunds.Add(contract.RenterFunds)
	}

	// Sum up total uploaded data
	for _, contract := range append(contracts.ActiveContracts, disabledContracts...) {
		metrics.ContractTotalSize += contract.Size
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
