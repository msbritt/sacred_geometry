package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var primeConstants = [][]int{
	{3, 5, 7},
	{11, 13, 17},
	{19, 23, 29},
	{31, 37, 41},
	{43, 47, 53},
	{59, 61, 67},
	{71, 73, 79},
	{83, 89, 97},
	{101, 103, 107},
}

const (
	maxSpellLevel = 9 // Maximum spell level possible in Pathfinder 1e
)

var (
	casterLevel = 6
	engineering = 6
)

type SpellRange struct {
	Type     string
	Distance int
}

type RangeCalculator struct {
	Touch  int
	Close  SpellRange
	Medium SpellRange
	Long   SpellRange
}

func NewRangeCalculator(casterLevel int) *RangeCalculator {
	return &RangeCalculator{
		Touch:  0,
		Close:  SpellRange{"close", 25 + (casterLevel/2)*5},
		Medium: SpellRange{"medium", 100 + casterLevel*10},
		Long:   SpellRange{"long", 400 + casterLevel*40},
	}
}

type Duration struct {
	Value   int
	Unit    string // "rounds", "minutes", "hours", etc.
	IsLevel bool   // true if duration scales with level
}

type DamageRoll struct {
	NumDice  int
	DiceType int  // e.g., 6 for d6
	Modifier int  // for things like Magic Missile's +1
	PerLevel bool // if damage scales with level
	MaxDice  int  // maximum number of dice (e.g., 5 for Shocking Grasp)
}

type Spell struct {
	Name           string
	BaseLevel      int
	Range          string
	DamageRoll     DamageRoll
	Duration       Duration
	MetamagicFeats []string
}

func rollDice(n int) []int {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	dice := make([]int, n)
	for i := range dice {
		dice[i] = r.Intn(6) + 1 // Generate a random number between 1 and 6 (inclusive).
	}
	return dice
}

func getPrimeConstants(level int) []int {
	return primeConstants[level-1]
}

func evalExpression(nums []int, ops []string) (int, string) {
	if len(nums) == 0 {
		return 0, ""
	}
	result := nums[0]
	expression := fmt.Sprintf("%d", nums[0])
	for i := 1; i < len(nums); i++ {
		nextNum := nums[i]
		nextOp := ops[i-1]
		previousOp := "+"
		if i > 1 {
			previousOp = ops[i-2]
		}
		if (previousOp == "+" || previousOp == "-") && (nextOp == "*" || nextOp == "/") {
			expression = fmt.Sprintf("(%s) %s %d", expression, nextOp, nextNum)
		} else {
			expression = fmt.Sprintf("%s %s %d", expression, nextOp, nextNum)
		}
		switch nextOp {
		case "+":
			result += nextNum
		case "-":
			result -= nextNum
		case "*":
			result *= nextNum
		case "/":
			if nextNum != 0 {
				result /= nextNum
			} else {
				return 0, ""
			}
		}
	}
	return result, expression
}

func findCombinationToPrime(dice []int, prime int) (string, bool) {
	operations := []string{"+", "-", "*", "/"}
	n := len(dice)
	for i := 1; i < (1 << uint(n)); i++ {
		var subset []int
		for j := 0; j < n; j++ {
			if i&(1<<uint(j)) != 0 {
				subset = append(subset, dice[j])
			}
		}
		perm := permutations(subset)
		for _, p := range perm {
			opsComb := combinations(len(p)-1, operations)
			for _, ops := range opsComb {
				result, expr := evalExpression(p, ops)
				if result == prime {
					return expr, true
				}
			}
		}
	}
	return "", false
}

func permutations(nums []int) [][]int {
	var helper func([]int, int)
	res := [][]int{}
	helper = func(arr []int, n int) {
		if n == 1 {
			tmp := make([]int, len(arr))
			copy(tmp, arr)
			res = append(res, tmp)
		} else {
			for i := 0; i < n; i++ {
				helper(arr, n-1)
				if n%2 == 1 {
					arr[0], arr[n-1] = arr[n-1], arr[0]
				} else {
					arr[i], arr[n-1] = arr[n-1], arr[i]
				}
			}
		}
	}
	helper(nums, len(nums))
	return res
}

func combinations(n int, elements []string) [][]string {
	if n == 0 {
		return [][]string{{}}
	}
	var result [][]string
	for _, e := range elements {
		for _, c := range combinations(n-1, elements) {
			result = append(result, append([]string{e}, c...))
		}
	}
	return result
}

type Result struct {
	Prime      int
	Expression string
	Found      bool
}

type MetamagicEffect struct {
	LevelIncrease int
	Description   string
	Apply         func(spell *Spell)
}

var MetamagicEffects = map[string]MetamagicEffect{
	"extend": {
		LevelIncrease: 1,
		Description:   "Doubles the duration of the spell",
		Apply: func(spell *Spell) {
			if spell.Duration.Value > 0 {
				spell.Duration.Value *= 2
			}
		},
	},
	"empower": {
		LevelIncrease: 2,
		Description:   "All variable, numeric effects are increased by half (50%)",
		Apply: func(spell *Spell) {
			// Note: We'll apply the 1.5x multiplier in the display logic
		},
	},
	"reach": {
		LevelIncrease: 1,
		Description:   "Can cast touch spells at close range, close range spells at medium range, and medium range spells at long range",
		Apply: func(spell *Spell) {
			switch spell.Range {
			case "touch":
				spell.Range = "close"
			case "close":
				spell.Range = "medium"
			case "medium":
				spell.Range = "long"
			}
		},
	},
	"intensified": {
		LevelIncrease: 1,
		Description:   "Adds 5 damage dice to spells with damage dice that scale with level",
		Apply: func(spell *Spell) {
			if spell.DamageRoll.PerLevel && spell.DamageRoll.MaxDice > 0 {
				beforeMax := spell.DamageRoll.MaxDice
				spell.DamageRoll.MaxDice += 5 // Add 5 to whatever the original max was
				fmt.Printf("Debug: Intensified increased MaxDice from %d to %d\n",
					beforeMax, spell.DamageRoll.MaxDice)
			}
		},
	},
}

func calculateSpellLevel(spell Spell) int {
	level := spell.BaseLevel

	for _, metamagic := range spell.MetamagicFeats {
		if effect, exists := MetamagicEffects[strings.ToLower(metamagic)]; exists {
			level += effect.LevelIncrease
		}
	}

	return level
}

func applyMetamagicEffects(spell *Spell) {
	for _, metamagic := range spell.MetamagicFeats {
		if effect, exists := MetamagicEffects[strings.ToLower(metamagic)]; exists {
			effect.Apply(spell)
		}
	}
}

// Helper function to format duration
func formatDuration(d Duration, casterLevel int) string {
	value := d.Value
	if d.IsLevel {
		value *= casterLevel
	}

	if value == 1 {
		// Remove trailing 's' for singular units
		return fmt.Sprintf("1 %s", strings.TrimSuffix(d.Unit, "s"))
	}
	return fmt.Sprintf("%d %s", value, d.Unit)
}

func mustAtoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

// Helper function to parse damage string (e.g., "1d6/level", "6d6", "1d4+1")
func parseDamage(dmgStr string) DamageRoll {
	if dmgStr == "" {
		return DamageRoll{}
	}

	var roll DamageRoll

	// Check for per-level scaling
	if strings.Contains(dmgStr, "/level") {
		roll.PerLevel = true
		dmgStr = strings.Replace(dmgStr, "/level", "", 1)
	}

	// Parse basic roll (e.g., "1d6" or "1d4+1")
	parts := strings.Split(dmgStr, "+")
	diceStr := parts[0]

	// Parse modifier if exists
	if len(parts) > 1 {
		roll.Modifier, _ = strconv.Atoi(parts[1])
	}

	// Parse dice (e.g., "1d6")
	diceParts := strings.Split(diceStr, "d")
	if len(diceParts) == 2 {
		roll.NumDice, _ = strconv.Atoi(diceParts[0])
		roll.DiceType, _ = strconv.Atoi(diceParts[1])
	}

	// Set default max dice for per-level spells
	if roll.PerLevel {
		roll.MaxDice = 5 // Default max for most spells
	}

	return roll
}

func formatDamage(roll DamageRoll, level int) string {
	if roll.NumDice == 0 {
		return ""
	}

	numDice := roll.NumDice
	if roll.PerLevel {
		numDice *= level
		// If the spell has a max and we're over it, cap at max
		if roll.MaxDice > 0 && numDice > roll.MaxDice {
			numDice = roll.MaxDice
		}
	}

	result := fmt.Sprintf("%dd%d", numDice, roll.DiceType)
	if roll.Modifier > 0 {
		result += fmt.Sprintf("+%d", roll.Modifier)
	}
	return result
}

// Helper function to parse duration string (e.g., "1 round", "1 minute/level")
func parseDuration(durStr string) Duration {
	if durStr == "" || durStr == "instantaneous" {
		return Duration{Value: 0, Unit: "", IsLevel: false}
	}

	var duration Duration

	// Check for per-level scaling
	if strings.Contains(durStr, "per_level") {
		duration.IsLevel = true
		durStr = strings.Replace(durStr, "per_level", "", 1)
	}

	// Parse value and unit
	parts := strings.Fields(durStr)
	if len(parts) >= 2 {
		duration.Value = mustAtoi(parts[0])
		duration.Unit = parts[1]
	}

	return duration
}

func readSpellsFromCSV(filename string) ([]Spell, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	var spells []Spell

	// Skip header row
	_, err = reader.Read()
	if err != nil {
		return nil, fmt.Errorf("error reading header: %v", err)
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading record: %v", err)
		}

		spell := Spell{
			Name:       record[0],
			BaseLevel:  mustAtoi(record[1]),
			Range:      record[3],
			DamageRoll: parseDamage(record[4]),
			Duration:   parseDuration(record[5]),
		}

		// Parse metamagic feats if present (now in column 6)
		if len(record) > 6 && record[6] != "" {
			spell.MetamagicFeats = strings.Split(record[6], ";")
			if debugMode {
				fmt.Printf("Debug: Loaded metamagic feats for %s: %v\n", spell.Name, spell.MetamagicFeats)
			}
		}

		spells = append(spells, spell)
	}

	return spells, nil
}

var debugMode bool
var verboseMode bool

func main() {
	// Check for debug and verbose flags
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--debug":
			debugMode = true
		case "--verbose":
			verboseMode = true
		}
	}

	// Read spells from spells.csv
	spells, err := readSpellsFromCSV("spells.csv")
	if err != nil {
		fmt.Printf("Error reading spells.csv: %v\n", err)
		return
	}

	// Process each spell
	for _, spell := range spells {
		spellCopy := spell // Create a copy to modify
		if debugMode {
			fmt.Printf("Debug: Initial copy - MaxDice: %d\n", spellCopy.DamageRoll.MaxDice)
		}

		spellLevel := calculateSpellLevel(spellCopy)
		if spellLevel > maxSpellLevel {
			fmt.Printf("Spell %s with metamagic exceeds maximum spell level (%d)\n", spellCopy.Name, maxSpellLevel)
			continue
		}

		// Set caster level for damage and range calculations
		casterLevel = spellLevel

		// Initialize range calculator
		ranges := NewRangeCalculator(casterLevel)

		if debugMode {
			fmt.Printf("Debug: Before applyMetamagicEffects - MaxDice: %d\n", spellCopy.DamageRoll.MaxDice)
		}
		applyMetamagicEffects(&spellCopy)
		if debugMode {
			fmt.Printf("Debug: After applyMetamagicEffects - MaxDice: %d\n", spellCopy.DamageRoll.MaxDice)
		}

		primes := getPrimeConstants(spellLevel)
		dice := rollDice(engineering)

		// Build metamagic string with level increases
		var metamagicParts []string
		for _, feat := range spell.MetamagicFeats {
			if effect, exists := MetamagicEffects[strings.ToLower(feat)]; exists {
				metamagicParts = append(metamagicParts, fmt.Sprintf("%s: +%d", feat, effect.LevelIncrease))
			}
		}

		metamagicStr := fmt.Sprintf("Base Level %d", spell.BaseLevel)
		if len(metamagicParts) > 0 {
			metamagicStr += fmt.Sprintf("; metamagic adjustments — %s; Final Spell Level: %d",
				strings.Join(metamagicParts, ", "), spellLevel)
		}

		fmt.Printf("\nCalculating %s: %s\n", spellCopy.Name, metamagicStr)
		fmt.Printf("Prime constants for modified spell level %d: %v\n", spellLevel, primes)
		fmt.Printf("Rolling %d d6 dice: %v\n", engineering, dice)

		// Use existing concurrent prime calculation logic
		var wg sync.WaitGroup
		resultChan := make(chan Result, len(primes))

		for _, prime := range primes {
			wg.Add(1)
			go func(p int) {
				defer wg.Done()
				expr, found := findCombinationToPrime(dice, p)
				resultChan <- Result{Prime: p, Expression: expr, Found: found}
			}(prime)
		}

		wg.Wait()
		close(resultChan)

		success := true
		var expressions []Result
		for result := range resultChan {
			if !result.Found {
				success = false
				break
			}
			expressions = append(expressions, result)
		}

		if success {
			fmt.Printf("\nSuccess: Sacred Geometry succeeded for %s!\n", spellCopy.Name)

			// Display the mathematical expressions for each prime only in verbose mode
			if verboseMode {
				fmt.Printf("\nPrime number calculations:\n")
				for _, result := range expressions {
					fmt.Printf("- %d = %s\n", result.Prime, result.Expression)
				}
			}

			// Display metamagic effects
			for _, metamagic := range spellCopy.MetamagicFeats {
				if effect, exists := MetamagicEffects[strings.ToLower(metamagic)]; exists {
					fmt.Printf("- %s: %s\n", metamagic, effect.Description)
				}
			}

			// Display range
			switch spellCopy.Range {
			case "touch":
				fmt.Printf("Range: Touch\n")
			case "close":
				fmt.Printf("Range: %d ft\n", ranges.Close.Distance)
			case "medium":
				fmt.Printf("Range: %d ft\n", ranges.Medium.Distance)
			case "long":
				fmt.Printf("Range: %d ft\n", ranges.Long.Distance)
			}

			// Display damage with all modifications
			if spellCopy.DamageRoll.NumDice > 0 {
				baseDamage := formatDamage(spell.DamageRoll, casterLevel)
				fmt.Printf("Base Damage: %s\n", baseDamage)

				// Show Intensified damage if present
				for _, metamagic := range spellCopy.MetamagicFeats {
					if strings.ToLower(metamagic) == "intensified" {
						// Calculate actual dice after intensified
						intensifiedDamage := formatDamage(spellCopy.DamageRoll, casterLevel)
						actualDice := spellCopy.DamageRoll.NumDice * casterLevel
						if actualDice > spellCopy.DamageRoll.MaxDice {
							actualDice = spellCopy.DamageRoll.MaxDice
						}
						fmt.Printf("Intensified Damage: %s (max dice increased to %d)\n",
							intensifiedDamage, spellCopy.DamageRoll.MaxDice)
						break
					}
				}

				// Show Empowered damage if present
				for _, metamagic := range spellCopy.MetamagicFeats {
					if strings.ToLower(metamagic) == "empower" {
						empoweredDamage := formatDamage(spellCopy.DamageRoll, casterLevel)
						fmt.Printf("Empowered Damage: %s (×1.5)\n", empoweredDamage)
						break
					}
				}
			}

			if spellCopy.Duration.Value > 0 {
				fmt.Printf("Duration: %s\n", formatDuration(spellCopy.Duration, casterLevel))
			}
		} else {
			fmt.Printf("\nFailure: Sacred Geometry failed for %s\n", spellCopy.Name)
		}
	}
}
