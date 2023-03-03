package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fentec-project/gofe/abe"
)

type Circuit struct {
	leftOperand  string
	operator     string
	rightOperand string
}

type Config struct {
	NumMagnitude     int
	AttributesNeeded int
}

func ParseFlags() (Config, error) {
	config := Config{}
	flag.IntVar(&config.NumMagnitude, "m", 6, "Define the magnitude of the abe policy circuit(reduction)")
	flag.IntVar(&config.AttributesNeeded, "n", 5, "Sets the number of nodes in the system(reduction)")

	flag.IntVar(&config.NumMagnitude, "magnitude", 6, "Define the magnitude of the abe policy circuit")
	flag.IntVar(&config.AttributesNeeded, "numofports", 5, "Sets the number of nodes in the system")
	flag.Parse()

	if config.NumMagnitude <= 0 || config.NumMagnitude >= 100 {
		config.NumMagnitude = 10
	}

	if config.AttributesNeeded <= 0 || config.AttributesNeeded > 20 {
		config.AttributesNeeded = 20
	}

	return config, nil
}

// Randomly generate gate logic statements
func GenerateCircuits(numCircuits int) []Circuit {
	var circuits []Circuit
	for i := 0; i < numCircuits; i++ {
		left := strconv.Itoa(rand.Intn(10000))
		right := strconv.Itoa(rand.Intn(10000))
		operator := ""
		switch rand.Intn(2) {
		case 0:
			operator = "AND"
		case 1:
			operator = "OR"
		}
		circuits = append(circuits, Circuit{left, operator, right})
	}
	return circuits
}

// Converts gate logic statements to strings
func (c Circuit) String() string {
	return fmt.Sprintf("(%s %s %s)", c.leftOperand, c.operator, c.rightOperand)
}

// Converts gate logic statements to strings
func GenerateAttributes() []string {
	var attributes []string
	for i := 0; i < rand.Intn(7)+2; i++ {
		length := rand.Intn(8) + 1
		letters := make([]rune, length)
		for j := range letters {
			letters[j] = rune(rand.Intn(26) + 97)
		}
		attributes = append(attributes, string(letters))
	}
	return attributes
}

// Converts gate logic statements to strings
func GenerateLogicOperator() string {
	operators := []string{"AND", "OR"}
	return operators[rand.Intn(len(operators))]
}

func PolicySetup(num int) string {
	// Converts gate logic statements to strings
	numCircuits := num

	// Randomly generate gate logic statements
	circuits := GenerateCircuits(numCircuits)

	// Randomly generate gate logic statements
	attributes := GenerateAttributes()

	// Assemble gate logic statements and attribute content
	var parts []string
	for _, circuit := range circuits {
		parts = append(parts, circuit.String())
	}
	parts = append(parts, strings.Join(attributes, " "+GenerateLogicOperator()+" "))
	finalStr := strings.Join(parts, " AND ")

	return finalStr
}

// Resolves out property names and Boolean operators
func evaluateLogicCircuit(circuit string, numMax int) [][]string {

	properties := make([][]string, 0)
	for _, condition := range strings.Split(circuit, " AND ") {
		props := make([]string, 0)
		for _, p := range strings.Split(condition, " OR ") {
			props = append(props, p)
		}
		properties = append(properties, props)
	}

	cartesianProduct := make([][]string, 1)
	for _, props := range properties {
		if len(props) > 1 {
			tmp := make([][]string, 0)
			for _, prop := range props {
				for _, cp := range cartesianProduct {
					newCP := append([]string{prop}, cp...)
					tmp = append(tmp, newCP)
				}
			}
			cartesianProduct = tmp
		} else {
			for i := range cartesianProduct {
				cartesianProduct[i] = append(cartesianProduct[i], props[0])
			}
		}
	}

	// Resolves out property names and Boolean operators
	results := make([][]string, 0)
	i := 0
	for _, props := range cartesianProduct {
		isMatched := true
		for _, condition := range properties {
			isConditionMatched := false
			for _, p := range condition {
				if containsString(props, p) {
					isConditionMatched = true
					break
				}
			}
			if !isConditionMatched {
				isMatched = false
				break
			}
		}
		if isMatched {
			results = append(results, props)
			i++

			if i >= numMax {
				return results
			}
		}
	}

	return results
}

func containsString(strings []string, s string) bool {
	for _, str := range strings {
		if str == s {
			return true
		}
	}
	return false
}

func main() {
	rand.Seed(time.Now().Unix())
	help := flag.Bool("h", false, "Display Help")
	config, err := ParseFlags()
	if err != nil {
		panic(err)
	}
	if *help {
		fmt.Println("Use this program to initialize the abe system")
		fmt.Println()
		flag.PrintDefaults()
		return
	}
	numMagnitude := config.NumMagnitude
	AttributesNeeded := config.AttributesNeeded
	relay := abe.NewFAME()
	// Generate an ABE key pair
	mpk, msk, err := relay.GenerateMasterKeys()
	if err != nil {
		panic(err)
	}
	policy := PolicySetup(numMagnitude)
	msp, _ := abe.BooleanToMSP(policy, false) // The MSP structure defining the policy
	// Convert the key pair to JSON format and write it to a file
	mskJSON, _ := json.Marshal(msk)
	mpkJSON, _ := json.Marshal(mpk)
	mspJSON, _ := json.Marshal(msp)

	file, err := os.Create(".ABEenv")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	qualifiedAttributes := evaluateLogicCircuit(policy, AttributesNeeded)

	for _, subArr := range qualifiedAttributes {
		for i, str := range subArr {
			str = strings.ReplaceAll(str, "(", "")
			str = strings.ReplaceAll(str, ")", "")
			subArr[i] = str
		}
	}

	_, err = fmt.Fprintf(file, "MSK=%s\nMPK=%s\nMSP=%s\nPolicy=%s\nQualifiedAttributes=%s", mskJSON, mpkJSON, mspJSON, policy, qualifiedAttributes)
	if err != nil {
		panic(err)
	}

	for i, qualifiedAttribute := range qualifiedAttributes {
		keys, _ := relay.GenerateAttribKeys(qualifiedAttribute, msk)
		keysJSON, err := json.Marshal(keys)
		if err != nil {
			panic(err)
		}
		MemberEnv := fmt.Sprintf("Member%dEnv", i)
		file, err := os.Create(MemberEnv)
		if err != nil {
			fmt.Println(err)
			return
		}

		_, err = fmt.Fprintf(file, "MPK=%s\nSecKeys=%s", mpkJSON, keysJSON)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	fmt.Println("Initialization successful!")
}
