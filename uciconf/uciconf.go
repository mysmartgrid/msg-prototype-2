package uciconf

import(
	"os"
	"bufio"
	"regexp"
	"strconv"
//	"fmt"
)

var configData map[string]map[string]map[string]string
var re1 *regexp.Regexp;
var re2 *regexp.Regexp;
var re3 *regexp.Regexp;

func init() {
	configData = make(map[string]map[string]map[string]string)
	re1 = regexp.MustCompile("^\\s*([\\w-]*)\\s* \\s*(.*)\\s*")
	re2 = regexp.MustCompile("^\\s*([\\w-]*)\\s* \\s*([\\w-]*)\\s* '\\s*(.*)\\s*'")
	re3 = regexp.MustCompile("^\\s*([\\w-]*)\\s* \\s*([\\w-]*)\\s* \\s*(.*)\\s*")
	//re3 = regexp.MustCompile("^\\s*([\\w-]*)\\s* \\s*([\\w-]*)\\s* \\s*([\\w-]*)\\s* \\s*(.*)\\s*")
}

func Get(namespace string, setting string, option string) string {
	namespaceMap := fetchNamespace(namespace)
	val, _ := namespaceMap[setting][option]
	return val
}

/*
func Get(namespace string, setting string) map[string]string {
	namespaceMap := fetchNamespace(namespace)
	val, _ := namespaceMap[setting]
	return val
}
*/

func GetUint(namespace string, setting string, option string) uint64 {
	namespaceMap := fetchNamespace(namespace)
	val, _ := namespaceMap[setting][option]
	parsedVal, _ := strconv.ParseUint(val, 10, 64)
	return parsedVal
}

/*
func GetInt(namespace string, setting string) int64 {
	namespaceMap := fetchNamespace(namespace)
	val, _ := namespaceMap[setting]
	parsedVal, _ := strconv.ParseInt(val, 10, 64)
	return parsedVal
}

func GetFloat(namespace string, setting string) float64 {
	namespaceMap := fetchNamespace(namespace)
	val, _ := namespaceMap[setting]
	parsedVal, _ := strconv.ParseFloat(val, 64)
	return parsedVal
}

func GetBool(namespace string, setting string) bool {
	namespaceMap := fetchNamespace(namespace)
	val, _ := namespaceMap[setting]
	parsedVal, _ := strconv.ParseBool(val)
	return parsedVal
}
*/

/*
func Copy(namespace string) map[string]string {
	namespaceMap := fetchNamespace(namespace)
	mapCopy := make(map[string]string)
	for k,v := range namespaceMap {
	  mapCopy[k] = v
	}
	return mapCopy
}
*/

func Set(namespace string, setting string, option string, value string) {
	namespaceMap := fetchNamespace(namespace)
	namespaceMap[setting][option] = value
}

func fetchNamespace(namespace string) map[string]map[string]string {
	namespaceMap, ok := configData[namespace]
	if !ok {
		importSettingsFromFile(namespace)
		namespaceMap, _ = configData[namespace]
	}
	return namespaceMap
}

func importSettingsFromFile(namespace string) error {
        var section string
	configData[namespace] = make(map[string]map[string]string)
	file, err := os.Open("/etc/config/"+ namespace )
	defer file.Close()
	if err != nil {
		// if no config file, that is fine and dandy, can still use it without config files.
		return err
	}
	reader := bufio.NewReader(file)
	scanner := bufio.NewScanner(reader)

	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		parsedLine := re2.FindStringSubmatch(line)
		if(len(parsedLine) == 0) {
			parsedLine = re3.FindStringSubmatch(line)
		}
		//fmt.Printf("Line: %d - '%s'\n", len(parsedLine), line)
		if(len(parsedLine) == 4) {
			//fmt.Printf("==>'%s' - '%s' - '%s'\n",
			//			parsedLine[1], parsedLine[2], parsedLine[3])
			if(parsedLine[1] == "config") {
				section = parsedLine[2] + "." + parsedLine[3]
				configData[namespace][section] = make(map[string]string)
				configData[namespace][section][parsedLine[2]] = parsedLine[3]
			} else {
			if(parsedLine[1] == "option") {
				configData[namespace][section][parsedLine[2]] = parsedLine[3]
			}
			}
		} else {
		  parsedLine := re1.FindStringSubmatch(line)
		  if(len(parsedLine) == 3) {
			if(parsedLine[1] == "config") {
				section = parsedLine[2]
				configData[namespace][section] = make(map[string]string)
			}
		  }
		}
	}
	file.Close()
	return nil
}
