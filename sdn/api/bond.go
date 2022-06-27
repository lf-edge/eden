package api

import (
	"bytes"
	"encoding/json"
)

// Bond is aggregating multiple ports for load-sharing and redundancy purposes.
// Also known as Link aggregation group (LAG).
type Bond struct {
	// LogicalLabel : logical name used for reference.
	LogicalLabel string
	// IfName : interface name in the kernel.
	IfName string
	// Ports : list of aggregated ports, referenced by logical labels.
	Ports []string
	// Mode : bonding policy.
	Mode BondMode
	// LacpRate : LACPDU packets transmission rate.
	// Applicable for BondMode802Dot3AD only.
	LacpRate LacpRate
	// MIIMonitor : MII link state monitoring.
	// Link monitoring is either disabled or one of the monitors
	// is enabled (MII or ARP), never both at the same time.
	MIIMonitor BondMIIMonitor
	// ARPMonitor : ARP-based link state monitoring.
	// Link monitoring is either disabled or one of the monitors
	// is enabled (MII or ARP), never both at the same time.
	ARPMonitor BondArpMonitor
}

// BondMIIMonitor : MII link monitoring parameters.
type BondMIIMonitor struct {
	// Enabled : set to true to enable MII.
	Enabled bool
	// Interval specifies the MII link monitoring frequency in milliseconds.
	// This determines how often the link state of each bond slave is inspected
	// for link failures.
	Interval uint32
	// UpDelay specifies the time, in milliseconds, to wait before enabling
	// a bond slave after a link recovery has been detected.
	// The UpDelay value should be a multiple of the monitoring interval; if not,
	// it will be rounded down to the nearest multiple.
	// The default value is 0.
	UpDelay uint32
	// DownDelay specifies the time, in milliseconds, to wait before disabling a bond
	// slave after a link failure has been detected.
	// The DownDelay value should be a multiple of the monitoring interval; if not,
	// it will be rounded down to the nearest multiple.
	// The default value is 0.
	DownDelay uint32
}

// BondArpMonitor : ARP-based link monitoring parameters.
type BondArpMonitor struct {
	// Enabled : set to true to enable ARP-based link monitoring.
	Enabled bool
	// Interval specifies the ARP link monitoring frequency in milliseconds.
	Interval uint32
	// IPTargets specifies the IPv4 addresses to use as ARP monitoring peers.
	// These are the targets of ARP requests sent to determine the health of links.
	IPTargets []string
}

// LacpRate specifies the rate in which bond driver will ask LACP link partners
// to transmit LACPDU packets in 802.3ad mode.
type LacpRate uint8

const (
	// LacpRateSlow : Request partner to transmit LACPDUs every 30 seconds.
	// This is the default rate.
	LacpRateSlow LacpRate = iota
	// LacpRateFast : Request partner to transmit LACPDUs every 1 second.
	LacpRateFast
)

// LacpRateToString : convert LacpRate to string representation used in JSON.
var LacpRateToString = map[LacpRate]string{
	LacpRateSlow: "slow",
	LacpRateFast: "fast",
}

// LacpRateToID : get LacpRate from a string representation.
var LacpRateToID = map[string]LacpRate{
	"":     LacpRateSlow,
	"slow": LacpRateSlow,
	"fast": LacpRateFast,
}

// MarshalJSON marshals the enum as a quoted json string.
func (s LacpRate) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(LacpRateToString[s])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON un-marshals a quoted json string to the enum value.
func (s *LacpRate) UnmarshalJSON(b []byte) error {
	var j string
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	*s = LacpRateToID[j]
	return nil
}

// BondMode specifies the policy indicating how bonding slaves are used
// during network transmissions.
type BondMode uint8

const (
	// BondModeBalanceRR : Round-Robin
	// This is the default mode.
	BondModeBalanceRR BondMode = iota
	// BondModeActiveBackup : Active/Backup
	BondModeActiveBackup
	// BondModeBalanceXOR : select slave for a packet using a hash function
	BondModeBalanceXOR
	// BondModeBroadcast : send every packet on all slaves
	BondModeBroadcast
	// BondMode802Dot3AD : IEEE 802.3ad Dynamic link aggregation
	BondMode802Dot3AD
	// BondModeBalanceTLB : Adaptive transmit load balancing
	BondModeBalanceTLB
	// BondModeBalanceALB : Adaptive load balancing
	BondModeBalanceALB
)

// BondModeToString : convert BondMode to string representation used in JSON.
var BondModeToString = map[BondMode]string{
	BondModeBalanceRR:    "balance-rr",
	BondModeActiveBackup: "active-backup",
	BondModeBalanceXOR:   "balance-xor",
	BondModeBroadcast:    "broadcast",
	BondMode802Dot3AD:    "802.3ad",
	BondModeBalanceTLB:   "balance-tlb",
	BondModeBalanceALB:   "balance-alb",
}

// BondModeToID : get BondMode from a string representation.
var BondModeToID = map[string]BondMode{
	"":              BondModeBalanceRR,
	"balance-rr":    BondModeBalanceRR,
	"active-backup": BondModeActiveBackup,
	"balance-xor":   BondModeBalanceXOR,
	"broadcast":     BondModeBroadcast,
	"802.3ad":       BondMode802Dot3AD,
	"balance-tlb":   BondModeBalanceTLB,
	"balance-alb":   BondModeBalanceALB,
}

// MarshalJSON marshals the enum as a quoted json string.
func (s BondMode) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(BondModeToString[s])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON un-marshals a quoted json string to the enum value.
func (s *BondMode) UnmarshalJSON(b []byte) error {
	var j string
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	*s = BondModeToID[j]
	return nil
}
