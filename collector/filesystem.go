package collector

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"regexp"
	"strings"
)

// 忽略的挂载点和系统类型
const (
	defIgnoredMountPoints = "^/(dev|proc|sys|var/lib/docker/.+)($|/)"
	defIgnoredFSTypes     = "^(autofs|binfmt_misc|bpf|cgroup2?|configfs|debugfs|devpts|devtmpfs|tmpfs|fusectl|hugetlbfs|iso9660|mqueue|nsfs|overlay|proc|procfs|pstore|rpc_pipefs|securityfs|selinuxfs|squashfs|sysfs|tracefs)$"
	filesystem            = "filesystem_info"
)

// 标签结构体
type filesystemLabels struct {
	device     string
	mountPoint string
	fsType     string
	//options string
}

// 总的文件系统结构体
type filesystemStats struct {
	labels            filesystemLabels
	size, free, avail float64
	files, filesFree  float64
	//ro, deviceError   float64
}

type ScrapeFilesystemInfo struct{}

// Name method of Scraper
func (ScrapeFilesystemInfo) Name() string {
	return filesystem
}

// Help method of Scraper
func (ScrapeFilesystemInfo) Help() string {
	return "Scrape filesystem like mountpoint,size,freesize,devicename and so on..."
}

// Version method of Scraper
func (ScrapeFilesystemInfo) Version() float64 {
	return 1.0
}

// Scrape method of Scraper
func (ScrapeFilesystemInfo) Scrape(ch chan<- prometheus.Metric, logger log.Logger) error {
	filesystemstats, err := getStats()
	if err != nil {
		return err
	}
	for _, info := range filesystemstats {
		newdesc := prometheus.NewDesc(
			prometheus.BuildFQName(namespace, filesystem, "total_free"),
			"Seconds the filesystem free in each mode.",
			[]string{"device", "mountpoint"}, nil,
		)
		ch <- prometheus.MustNewConstMetric(newdesc, prometheus.GaugeValue, info.free, info.labels.device, info.labels.mountPoint)
	}
	return nil
}

// step 1
// 获取挂载点详细信息，返回filesystemLabels切片，error
func mountPointDetails() ([]filesystemLabels, error) {
	file, err := os.Open(procFilePath("1/mounts"))
	if errors.Is(err, os.ErrNotExist) {
		// 两个挂在点,root 挂载点/proc/mounts 系统挂载点/proc/1/moounts
		fmt.Println("Reading root mouts failed,falling back to system mounts", err)
		file, err = os.Open(procFilePath("mounts"))
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return parseFilesystemLabels(file)
}

// step2
// 解析打开的文件系统内容返回filesystemLables
func parseFilesystemLabels(r io.Reader) ([]filesystemLabels, error) {
	var filesystems []filesystemLabels
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) < 4 {
			return nil, fmt.Errorf("malformed mount point information: %q", scanner.Text())
		}
		//fmt.Println(strings.Repeat(">", 40))
		//fmt.Println(parts)
		//fmt.Println(strings.Repeat(">", 40))
		// Ensure we handle the translation of \040 and \011
		// as per fstab(5).
		parts[1] = strings.Replace(parts[1], "\\040", " ", -1)
		parts[1] = strings.Replace(parts[1], "\\011", "\t", -1)

		filesystems = append(filesystems, filesystemLabels{
			device:     parts[0],
			mountPoint: rootfsStripPrefix(parts[1]),
			fsType:     parts[2],
			//options:    parts[3],
		})
	}
	return filesystems, scanner.Err()
}

// step3
// 实际抓取文件系统状态的函数，返回filesystemStats，供Scraper遍历发送chan
func getStats() ([]filesystemStats, error) {
	mps, err := mountPointDetails()
	if err != nil {
		fmt.Println(err.Error())
	}
	stats := []filesystemStats{}
	for _, labels := range mps {
		if ok, _ := regexp.MatchString(defIgnoredMountPoints, labels.mountPoint); ok {
			//fmt.Println("已忽略此挂载点", labels.mountPoint)
			continue
		}
		if ok, _ := regexp.MatchString(defIgnoredFSTypes, labels.fsType); ok {
			//fmt.Println("已忽略此类型", labels.fsType)
			continue
		}
		fmt.Printf("device:%s, mountpoiont:%s, fstype:%s\n", labels.device, labels.mountPoint, labels.fsType)
		buf := new(unix.Statfs_t)
		err = unix.Statfs(rootfsFilePath(labels.mountPoint), buf)
		//var ro float64
		//for _, option := range strings.Split(labels.options, ",") {
		//	if option == "ro" {
		//		ro = 1
		//		break
		//	}
		//}
		stats = append(stats, filesystemStats{
			labels:    labels,
			size:      float64(buf.Blocks) * float64(buf.Bsize),
			free:      float64(buf.Bfree) * float64(buf.Bsize),
			avail:     float64(buf.Bavail) * float64(buf.Bsize),
			files:     float64(buf.Files),
			filesFree: float64(buf.Ffree),
			//ro:        ro,
		})
	}
	return stats, nil
}
