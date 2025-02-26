package device

import (
	"context"
	"net"
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"

	logging "github.com/ipfs/go-log/v2"
	"github.com/linguohua/titan/api"
	"github.com/linguohua/titan/blockstore"
	"github.com/linguohua/titan/build"
	"github.com/shirou/gopsutil/v3/mem"
)

var log = logging.Logger("device")

type Device struct {
	deviceID      string
	publicIP      string
	internalIP    string
	bandwidthUp   int64
	bandwidthDown int64
	blockstore    blockstore.BlockStore
}

func NewDevice(deviceID, publicIP, internalIP string, bandwidthUp, bandwidthDown int64, blockstore blockstore.BlockStore) *Device {
	device := &Device{
		deviceID:      deviceID,
		publicIP:      publicIP,
		internalIP:    internalIP,
		bandwidthUp:   bandwidthUp,
		bandwidthDown: bandwidthDown,
		blockstore:    blockstore,
	}

	return device
}
func (device *Device) DeviceInfo(ctx context.Context) (api.DevicesInfo, error) {
	info := api.DevicesInfo{}

	v, err := api.VersionForType(api.RunningNodeType)
	if err != nil {
		return info, err
	}

	version := api.APIVersion{
		Version:    build.UserVersion(),
		APIVersion: v,
	}

	info.DeviceId = device.deviceID

	name := device.deviceID
	if len(name) > 10 {
		info.DeviceName = name[0:10]
	}
	info.ExternalIp = device.publicIP
	info.SystemVersion = version.String()
	info.InternalIp = device.internalIP
	info.BandwidthDown = float64(device.bandwidthDown)
	info.BandwidthUp = float64(device.bandwidthUp)

	mac, err := getMacAddr(info.InternalIp)
	if err != nil {
		log.Errorf("DeviceInfo getMacAddr err:%s", err.Error())
		return api.DevicesInfo{}, err
	}

	info.MacLocation = mac

	vmStat, err := mem.VirtualMemory()
	if err != nil {
		log.Errorf("getMemory: %s", err.Error())
	}

	if vmStat != nil {
		info.MemoryUsage = vmStat.UsedPercent
		info.Memory = float64(vmStat.Total)
	}

	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		log.Errorf("getCpuInfo: %s", err.Error())
	}

	info.CpuUsage = cpuPercent[0]
	info.CPUCores, _ = cpu.Counts(false)

	blockStorePath := device.blockstore.GetPath()
	usageStat, err := disk.Usage(blockStorePath)
	if err != nil {
		log.Errorf("get disk usage stat error: %s", err)
		return api.DevicesInfo{}, err
	}

	info.DiskUsage = usageStat.UsedPercent
	info.DiskSpace = float64(usageStat.Total)

	absPath, err := filepath.Abs(blockStorePath)
	if err != nil {
		return api.DevicesInfo{}, err
	}

	partitionsStat, err := disk.Partitions(false)
	if err != nil {
		log.Errorf("get partitioin stat: %s", err)
		return api.DevicesInfo{}, err
	}

	for _, partition := range partitionsStat {
		if partition.Mountpoint != "/" &&
			len(absPath) >= len(partition.Mountpoint) &&
			absPath[0:len(partition.Mountpoint)] == partition.Mountpoint {
			info.IoSystem = partition.Fstype
			break
		}
	}

	return info, nil
}

func getMacAddr(ip string) (string, error) {
	ifas, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, ifa := range ifas {
		addrs, err := ifa.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			localAddr := addr.(*net.IPNet)
			localIP := strings.Split(localAddr.IP.String(), ":")[0]

			if localIP == ip {
				return ifa.HardwareAddr.String(), nil
			}
		}
	}
	return "", nil
}

func (device *Device) GetDeviceID() string {
	return device.deviceID
}

func (device *Device) SetBandwidthUp(bandwidthUp int64) {
	device.bandwidthUp = bandwidthUp
}

func (device *Device) GetBandwidthUp() int64 {
	return device.bandwidthUp
}

func (device *Device) GetBandwidthDown() int64 {
	return device.bandwidthDown
}

func (device *Device) SetExternaIP(externaIP string) {
	device.publicIP = externaIP
}

func (device *Device) GetExternaIP() string {
	return device.publicIP
}

func (device *Device) GetInternalIP() string {
	return device.internalIP
}
