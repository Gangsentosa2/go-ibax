/*---------------------------------------------------------------------------------------------
 *  Copyright (c) IBAX. All rights reserved.
 *  See LICENSE in the project root for license information.
 *--------------------------------------------------------------------------------------------*/

package sqldb

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/IBAX-io/go-ibax/packages/converter"
)

// Key is model
type Key struct {
	ecosystem    int64
	accountKeyID int64 `gorm:"-"`

	ID        int64  `gorm:"primary_key;not null"`
	AccountID string `gorm:"column:account;not null"`
	PublicKey []byte `gorm:"column:pub;not null"`

	// Amount string `gorm:"not null"`
	// TODO UTXO balance replace Amount
	Balance decimal.Decimal `gorm:"-"`

	Maxpay  string `gorm:"not null"`
	Deleted int64  `gorm:"not null"`
	Blocked int64  `gorm:"not null"`
}

// SetTablePrefix is setting table prefix
func (m *Key) SetTablePrefix(prefix int64) *Key {
	m.ecosystem = prefix
	return m
}

// TableName returns name of table
func (m Key) TableName() string {
	if m.ecosystem == 0 {
		m.ecosystem = 1
	}
	return `1_keys`
}
func (m *Key) Disable() bool {
	return m.Deleted != 0 || m.Blocked != 0
}

// The GetBalance method must be called before it can be called
func (m *Key) CapableAmount() decimal.Decimal {
	amount := decimal.Zero
	if m.Balance.GreaterThan(decimal.Zero) {
		amount = m.Balance
	}
	maxpay := decimal.Zero
	if len(m.Maxpay) > 0 {
		maxpay, _ = decimal.NewFromString(m.Maxpay)
	}
	if maxpay.GreaterThan(decimal.Zero) && maxpay.LessThan(amount) {
		amount = maxpay
	}
	return amount
}

// Get is retrieving model from database
func (m *Key) Get(db *DbTransaction, wallet int64) (bool, error) {
	return isFound(GetDB(db).Where("id = ? and ecosystem = ?", wallet, m.ecosystem).First(m))
}

func (m *Key) GetAndBalance(db *DbTransaction, wallet int64) (bool, error) {
	found, err := m.Get(db, wallet)
	if found {
		txInputs, _ := GetTxOutputsEcosystem(db, m.ecosystem, []int64{wallet})
		totalAmount := decimal.Zero
		if len(txInputs) > 0 {
			for _, input := range txInputs {
				outputValue, _ := decimal.NewFromString(input.OutputValue)
				totalAmount = totalAmount.Add(outputValue)
			}
		}
		m.Balance = totalAmount
	}
	return found, err
}

// GetBalanceAndPut two methods Get and PutOutputsMap
func (m *Key) GetBalanceAndPut(db *DbTransaction, wallet int64, outputsMap map[int64][]SpentInfo) (bool, error) {
	found, err := m.Get(db, wallet)
	if found {
		balance := GetBalanceOutputsMap(m.ecosystem, wallet, outputsMap)
		if balance != nil {
			m.Balance = *balance
		} else {
			txInputs, _ := GetTxOutputsEcosystem(db, m.ecosystem, []int64{wallet})
			totalAmount := decimal.Zero
			if len(txInputs) > 0 {
				for _, input := range txInputs {
					outputValue, _ := decimal.NewFromString(input.OutputValue)
					totalAmount = totalAmount.Add(outputValue)
				}
				PutOutputsMap(m.ecosystem, wallet, txInputs, outputsMap)
			}
			m.Balance = totalAmount
		}
	}
	return found, err
}

func (m *Key) AccountKeyID() int64 {
	if m.accountKeyID == 0 {
		m.accountKeyID = converter.StringToAddress(m.AccountID)
	}
	return m.accountKeyID
}

// KeyTableName returns name of key table
func KeyTableName(prefix int64) string {
	return fmt.Sprintf("%d_keys", prefix)
}

// GetKeysCount returns common count of keys
func GetKeysCount() (int64, error) {
	var cnt int64
	row := DBConn.Raw(`SELECT count(*) key_count FROM "1_keys" WHERE ecosystem = 1`).Select("key_count").Row()
	err := row.Scan(&cnt)
	return cnt, err
}
