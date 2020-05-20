// // Usage example for this code...
//
// package libvirt
//
//import (
//	"fmt"
//	"os"
//	"os/exec"
//)
//
//func main() {
//	c := exec.Command("virsh", "domstats")
//	c.Env = append(os.Environ(), "LIBVIRT_DEFAULT_URI=qemu:///system")
//	out, err := c.CombinedOutput()
//	checkErr(err, "getting virsh output")
//
//	domains := ParseIntoStats(out)
//
//	for _, d := range domains {
//		for _, stat := range d.stats {
//			fmt.Printf(stat.InfluxString())
//		}
//	}
//}
//
package libvirt
