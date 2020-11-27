package pricers

type VirtualMachine struct {
	Size     string
	Location string
}

func (v *VirtualMachine) GetHourlyPrice() float64 {
	return 0.79
}
