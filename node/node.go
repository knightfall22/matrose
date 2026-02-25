package node

type Node struct {
	Name             string
	Ip               string
	Cores            int
	Memory           int
	MemoryAllocation int
	Disk             int
	DiskAllocation   int
	Role             string
	TaskCount        int
}
