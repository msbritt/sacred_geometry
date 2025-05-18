package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alexeyco/simpletable"
)

// ANSI color codes for terminal output
const (
	colorRed    = "\x1b[91m"
	colorGreen  = "\x1b[92m"
	colorYellow = "\x1b[93m"
	colorReset  = "\x1b[0m"
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
	casterLevel        = 6
	engineering        = 6
	TransmuterOfKorada = true // When true, Transmutation spells get +1 caster level
	showRanges         = flag.Bool("ranges", false, "Display the computed ranges for the current caster level")
	verbose            = flag.Bool("verbose", false, "Show detailed output")
	debug              = flag.Bool("debug", false, "Enable debug mode")
)

// RangeInfo holds a range's details and a function to compute its actual range.
type RangeInfo struct {
	Name        string
	Description string
	Compute     func(casterLevel int) string
}

// Define range data in a structured way
var rangeData = map[string]RangeInfo{
	"Touch": {
		Name:        "Touch",
		Description: "You must touch a creature or object to affect it.",
		Compute: func(casterLevel int) string {
			return "Touch range (no numerical distance)"
		},
	},
	"Close": {
		Name:        "Close",
		Description: "Spell reaches as far as 25 feet, plus an additional 5 feet for every 2 full caster levels.",
		Compute: func(casterLevel int) string {
			bonus := (casterLevel / 2) * 5
			total := 25 + bonus
			return fmt.Sprintf("%d feet (Base: 25 ft + Bonus: %d ft)", total, bonus)
		},
	},
	"Medium": {
		Name:        "Medium",
		Description: "Spell reaches as far as 100 feet plus 10 feet per caster level.",
		Compute: func(casterLevel int) string {
			total := 100 + 10*casterLevel
			return fmt.Sprintf("%d feet (Base: 100 ft + %d ft from caster level)", total, 10*casterLevel)
		},
	},
	"Long": {
		Name:        "Long",
		Description: "Spell reaches as far as 400 feet plus 40 feet per caster level.",
		Compute: func(casterLevel int) string {
			total := 400 + 40*casterLevel
			return fmt.Sprintf("%d feet (Base: 400 ft + %d ft from caster level)", total, 40*casterLevel)
		},
	},
	"Unlimited": {
		Name:        "Unlimited",
		Description: "Spell reaches anywhere on the same plane of existence.",
		Compute: func(casterLevel int) string {
			return "Unlimited range"
		},
	},
	"Personal": {
		Name:        "Personal",
		Description: "Spell affects only the caster.",
		Compute: func(casterLevel int) string {
			return "Personal (self only)"
		},
	},
}

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
	NumDice     int
	DiceType    int  // e.g., 6 for d6
	Modifier    int  // for things like Magic Missile's +1
	PerLevel    bool // if damage scales with level
	MaxDice     int  // maximum number of dice (e.g., 5 for Shocking Grasp)
	Projectiles int  // for spells like Magic Missile that have multiple projectiles
}

type Spell struct {
	Name           string
	BaseLevel      int
	School         string
	Range          string
	DamageRoll     DamageRoll
	Duration       Duration
	MetamagicFeats []string
	MetamagicMods  []string // Added for Lorandir's trait handling
}

// PrimeResult represents the result of finding a prime number combination
type PrimeResult struct {
	Prime      int
	Expression string
	Found      bool
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
			if (spell.DamageRoll.PerLevel && spell.DamageRoll.MaxDice > 0) ||
				(spell.Name == "Fireball" && spell.DamageRoll.MaxDice > 0) {
				beforeMax := spell.DamageRoll.MaxDice
				spell.DamageRoll.MaxDice += 5 // Add 5 to whatever the original max was
				if debugMode {
					fmt.Printf("Debug: Intensified increased MaxDice from %d to %d\n",
						beforeMax, spell.DamageRoll.MaxDice)
				}
			}
		},
	},
	"wayang_spell_hunter": {
		LevelIncrease: -1,
		Description:   "Lowers the total level of the spell by 1",
		Apply: func(spell *Spell) {
			// No additional effects to apply, the level reduction is handled in calculateSpellLevel
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

// Helper function to parse damage string (e.g., "1d6/level(max:5)", "6d6", "1d4+1")
func parseDamage(dmgStr string, spellName string) DamageRoll {
	if dmgStr == "" {
		return DamageRoll{}
	}

	var roll DamageRoll

	// Extract max dice if specified in the format (max:X)
	maxDiceMatch := false
	if strings.Contains(dmgStr, "(max:") {
		parts := strings.Split(dmgStr, "(max:")
		if len(parts) == 2 {
			maxPart := parts[1]
			maxPart = strings.TrimSuffix(maxPart, ")")
			maxDice, err := strconv.Atoi(maxPart)
			if err == nil {
				roll.MaxDice = maxDice
				maxDiceMatch = true
			}
			// Remove the max dice part from the damage string
			dmgStr = parts[0]
		}
	}

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

	// Set default max dice for per-level spells if not explicitly specified
	if roll.PerLevel && !maxDiceMatch {
		roll.MaxDice = 5 // Default max for most spells as a fallback
	}

	// Special handling for Magic Missile
	if spellName == "Magic Missile" {
		// Magic Missile always has 1 projectile at level 1, +1 for every 2 levels after that, max 5
		// This will be calculated at runtime based on caster level
		roll.Projectiles = 1 // Base value, will be adjusted based on caster level
	}

	return roll
}

func formatDamage(roll DamageRoll, level int, spellName string) string {
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

	// Special handling for Magic Missile
	if spellName == "Magic Missile" {
		// Calculate number of missiles based on caster level
		// 1 at level 1, +1 for every 2 levels after that, max 5
		projectiles := 1 + (level-1)/2
		if projectiles > 5 {
			projectiles = 5 // Maximum of 5 missiles
		}

		// Append the number of missiles to the damage string
		result += fmt.Sprintf(" (%d missiles)", projectiles)
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

// CommentFilterReader is a custom io.Reader that skips lines starting with a comment prefix
type CommentFilterReader struct {
	reader        *bufio.Reader
	commentPrefix string
	buffer        []byte
}

// Read implements the io.Reader interface
func (r *CommentFilterReader) Read(p []byte) (n int, err error) {
	if len(r.buffer) > 0 {
		// If we have data in the buffer, return it
		n = copy(p, r.buffer)
		r.buffer = r.buffer[n:]
		return n, nil
	}

	// Read a line
	for {
		line, err := r.reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return 0, err
		}

		// If we've reached EOF and have no data, return EOF
		if err == io.EOF && len(line) == 0 {
			return 0, io.EOF
		}

		// Skip comment lines
		if len(line) > 0 && len(r.commentPrefix) > 0 && strings.HasPrefix(strings.TrimSpace(string(line)), r.commentPrefix) {
			if debugMode {
				fmt.Printf("Debug: Skipping comment line: %s", string(line))
			}
			// If we've reached EOF, return it
			if err == io.EOF {
				return 0, io.EOF
			}
			continue
		}

		// Copy data to output and buffer any remaining
		n = copy(p, line)
		if n < len(line) {
			r.buffer = line[n:]
		}

		// If we've reached EOF, we'll return it on the next call
		if err == io.EOF && n > 0 {
			return n, nil
		}
		return n, err
	}
}

func readSpellsFromCSV(filename string) ([]Spell, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Create a custom reader that skips comment lines
	commentFilterReader := &CommentFilterReader{
		reader:        bufio.NewReader(file),
		commentPrefix: "#",
	}

	reader := csv.NewReader(commentFilterReader)
	var spells []Spell

	// Read header row
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("error reading header: %v", err)
	}

	// Find column indices
	colIdx := map[string]int{}
	for i, col := range header {
		colIdx[strings.ToLower(col)] = i
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
			Name:       record[colIdx["name"]],
			BaseLevel:  mustAtoi(record[colIdx["baselevel"]]),
			School:     record[colIdx["school"]],
			Range:      record[colIdx["range"]],
			DamageRoll: parseDamage(record[colIdx["damage"]], record[colIdx["name"]]),
			Duration:   parseDuration(record[colIdx["duration"]]),
		}

		// Parse new boolean metamagic columns
		var feats []string
		if idx, ok := colIdx["empower"]; ok && idx < len(record) && strings.EqualFold(record[idx], "Yes") {
			feats = append(feats, "empower")
		}
		if idx, ok := colIdx["intensified"]; ok && idx < len(record) && strings.EqualFold(record[idx], "Yes") {
			feats = append(feats, "intensified")
		}
		if idx, ok := colIdx["reach"]; ok && idx < len(record) && strings.EqualFold(record[idx], "Yes") {
			feats = append(feats, "reach")
		}
		if idx, ok := colIdx["extend"]; ok && idx < len(record) && strings.EqualFold(record[idx], "Yes") {
			feats = append(feats, "extend")
		}
		spell.MetamagicFeats = feats

		spells = append(spells, spell)
	}

	return spells, nil
}

var debugMode bool
var verboseMode bool

// formatTableOutput formats the spell information in a table format
func formatTableOutput(spell Spell, success bool, updatedRange string, diceCount int) string {
	// Check for metamagic feats
	hasEmpower := false
	hasIntensify := false
	for _, feat := range spell.MetamagicFeats {
		if strings.ToLower(feat) == "empower" {
			hasEmpower = true
		}
		if strings.ToLower(feat) == "intensified" {
			hasIntensify = true
		}
	}

	// Calculate the number of dice to be rolled based on the CSV value, caster level, and metamagic
	diceToRoll := diceCount
	if spell.DamageRoll.PerLevel {
		diceToRoll = spell.DamageRoll.NumDice * casterLevel
		if spell.DamageRoll.MaxDice > 0 && diceToRoll > spell.DamageRoll.MaxDice {
			diceToRoll = spell.DamageRoll.MaxDice
		}
		if hasIntensify {
			diceToRoll += 5
			if diceToRoll > casterLevel {
				diceToRoll = casterLevel
			}
		}
	}

	// If Empower is applied, multiply the dice count by 1.5
	if hasEmpower {
		diceToRoll = int(float64(diceToRoll) * 1.5)
	}

	// Format the dice count with die type and empower notation if applicable
	var diceStr string
	if diceToRoll > 0 {
		diceStr = fmt.Sprintf("%dd%d", diceToRoll, spell.DamageRoll.DiceType)
		if hasEmpower {
			diceStr += " (×1.5)"
		}
	}

	// Format boolean values as Yes/No
	empowerStr := "No"
	if hasEmpower {
		empowerStr = "Yes"
	}
	intensifyStr := "No"
	if hasIntensify {
		intensifyStr = "Yes"
	}

	return fmt.Sprintf("%-7s | %-20s | %-25s | %-15s | %-8s | %-10s",
		empowerStr,
		spell.Name,
		updatedRange,
		diceStr,
		intensifyStr,
		empowerStr)
}

func main() {
	flag.Parse()

	// If the ranges flag is set, display range information and exit
	if *showRanges {
		displayRangeInfo(casterLevel)
		return
	}

	// Read spells from CSV file
	spells, err := readSpellsFromCSV("spells.csv")
	if err != nil {
		fmt.Printf("Error reading spells.csv: %v\n", err)
		fmt.Println("Using default spell list...")
		// Use default spell list if CSV reading fails
		spells = []Spell{
			{
				Name:          "Bull's Strength",
				Duration:      Duration{Value: 1, Unit: "minute", IsLevel: true},
				Range:         "Touch",
				BaseLevel:     2,
				MetamagicMods: []string{},
			},
			{
				Name:          "Enlarge Person",
				Duration:      Duration{Value: 1, Unit: "minute", IsLevel: true},
				Range:         "Close",
				BaseLevel:     1,
				MetamagicMods: []string{},
			},
			{
				Name:          "Mage Armor",
				Duration:      Duration{Value: 1, Unit: "hour", IsLevel: true},
				Range:         "Touch",
				BaseLevel:     1,
				MetamagicMods: []string{},
			},
			{
				Name:          "Shocking Grasp",
				Duration:      Duration{Value: 0, Unit: "instantaneous", IsLevel: false},
				Range:         "Touch",
				BaseLevel:     1,
				MetamagicMods: []string{"Metamagic"},
			},
			{
				Name:          "Mirror Image",
				Duration:      Duration{Value: 1, Unit: "minute", IsLevel: true},
				Range:         "Personal",
				BaseLevel:     2,
				MetamagicMods: []string{},
			},
		}
	}

	// Process each spell
	fmt.Printf("Caster Level: %d\nEngineering: %d\n\n", casterLevel, engineering)

	if !*verbose {
		// Create a new table
		table := simpletable.New()

		// Set table header
		table.Header = &simpletable.Header{
			Cells: []*simpletable.Cell{
				{Align: simpletable.AlignLeft, Text: "Status"},
				{Align: simpletable.AlignLeft, Text: "Spell Name"},
				{Align: simpletable.AlignLeft, Text: "Updated Range"},
				{Align: simpletable.AlignLeft, Text: fmt.Sprintf("Dice Count: CL=%d", casterLevel)},
				{Align: simpletable.AlignLeft, Text: fmt.Sprintf("Dice Count: CL=%d", casterLevel+2)},
				{Align: simpletable.AlignLeft, Text: "Empower"},
				{Align: simpletable.AlignLeft, Text: "Intensify"},
			},
		}

		for _, spell := range spells {
			// Calculate effective spell level considering Lorandir's trait
			effectiveSpellLevel := spell.BaseLevel
			if len(spell.MetamagicMods) > 0 {
				effectiveSpellLevel = spell.BaseLevel - 1
				if *verbose {
					fmt.Printf("  Metamagic applied - Lorandir's trait reduces spell level by 1 (from %d to %d)\n",
						spell.BaseLevel, effectiveSpellLevel)
				}
			}
			// Ensure effectiveSpellLevel is at least 1
			if effectiveSpellLevel < 1 {
				effectiveSpellLevel = 1
			}

			// Get updated range
			updatedRange := spell.Range
			if info, ok := rangeData[spell.Range]; ok {
				updatedRange = info.Compute(casterLevel)
			}

			// Calculate dice count for current level
			diceCount := 0
			if spell.DamageRoll.NumDice > 0 {
				diceCount = spell.DamageRoll.NumDice
				if spell.DamageRoll.PerLevel {
					diceCount *= casterLevel
					if spell.DamageRoll.MaxDice > 0 && diceCount > spell.DamageRoll.MaxDice {
						diceCount = spell.DamageRoll.MaxDice
					}
				}
				// Apply Intensify if present
				for _, feat := range spell.MetamagicFeats {
					if strings.ToLower(feat) == "intensified" {
						diceCount += 5
						if diceCount > casterLevel {
							diceCount = casterLevel
						}
					}
				}
			}

			// Calculate dice count for caster level + 2
			diceCountPlus2 := 0
			if spell.DamageRoll.NumDice > 0 {
				diceCountPlus2 = spell.DamageRoll.NumDice
				if spell.DamageRoll.PerLevel {
					diceCountPlus2 *= (casterLevel + 2)
					if spell.DamageRoll.MaxDice > 0 && diceCountPlus2 > spell.DamageRoll.MaxDice {
						diceCountPlus2 = spell.DamageRoll.MaxDice
					}
				}
				// Apply Intensify if present
				for _, feat := range spell.MetamagicFeats {
					if strings.ToLower(feat) == "intensified" {
						diceCountPlus2 += 5
						if diceCountPlus2 > (casterLevel + 2) {
							diceCountPlus2 = (casterLevel + 2)
						}
					}
				}
			}

			// Format dice count for current level
			var diceStr string
			if diceCount > 0 {
				if spell.Name == "Magic Missile" {
					// Calculate number of missiles based on caster level
					// 1 at level 1, +1 for every 2 levels after that, max 5
					missiles := 1 + (casterLevel-1)/2
					if missiles > 5 {
						missiles = 5
					}
					// Multiply both dice and modifier by number of missiles
					totalDice := spell.DamageRoll.NumDice * missiles
					totalMod := spell.DamageRoll.Modifier * missiles
					if totalMod > 0 {
						diceStr = fmt.Sprintf("%dd%d + %d (%d missiles)", totalDice, spell.DamageRoll.DiceType, totalMod, missiles)
					} else {
						diceStr = fmt.Sprintf("%dd%d (%d missiles)", totalDice, spell.DamageRoll.DiceType, missiles)
					}
				} else {
					diceStr = fmt.Sprintf("%dd%d", diceCount, spell.DamageRoll.DiceType)
				}
				// Check for Empower metamagic
				for _, feat := range spell.MetamagicFeats {
					if strings.ToLower(feat) == "empower" {
						diceStr += " (×1.5)"
						break
					}
				}
			}

			// Format dice count for caster level + 2
			var diceStrPlus2 string
			if diceCountPlus2 > 0 {
				if spell.Name == "Magic Missile" {
					// Calculate number of missiles based on caster level + 2
					// 1 at level 1, +1 for every 2 levels after that, max 5
					missiles := 1 + ((casterLevel+2)-1)/2
					if missiles > 5 {
						missiles = 5
					}
					// Multiply both dice and modifier by number of missiles
					totalDice := spell.DamageRoll.NumDice * missiles
					totalMod := spell.DamageRoll.Modifier * missiles
					if totalMod > 0 {
						diceStrPlus2 = fmt.Sprintf("%dd%d + %d (%d missiles)", totalDice, spell.DamageRoll.DiceType, totalMod, missiles)
					} else {
						diceStrPlus2 = fmt.Sprintf("%dd%d (%d missiles)", totalDice, spell.DamageRoll.DiceType, missiles)
					}
				} else {
					diceStrPlus2 = fmt.Sprintf("%dd%d", diceCountPlus2, spell.DamageRoll.DiceType)
				}
				// Check for Empower metamagic
				for _, feat := range spell.MetamagicFeats {
					if strings.ToLower(feat) == "empower" {
						diceStrPlus2 += " (×1.5)"
						break
					}
				}
			}

			// Format boolean values
			empowerStr := "No"
			intensifyStr := "No"
			for _, feat := range spell.MetamagicFeats {
				if strings.ToLower(feat) == "empower" {
					empowerStr = "Yes"
				}
				if strings.ToLower(feat) == "intensified" {
					intensifyStr = "Yes"
				}
			}

			// Calculate Sacred Geometry
			primes := getPrimeConstants(effectiveSpellLevel)
			dice := rollDice(engineering)
			if *verbose {
				fmt.Printf("  Rolling %d d6: %v\n", engineering, dice)
			}

			var wg sync.WaitGroup
			results := make(chan PrimeResult, len(primes))

			for _, prime := range primes {
				wg.Add(1)
				go func(p int) {
					defer wg.Done()
					expr, found := findCombinationToPrime(dice, p)
					results <- PrimeResult{Prime: p, Expression: expr, Found: found}
				}(prime)
			}

			go func() {
				wg.Wait()
				close(results)
			}()

			success := true
			for result := range results {
				if result.Found {
					if *verbose {
						fmt.Printf("  %sPrime %d: %s%s\n", colorGreen, result.Prime, result.Expression, colorReset)
					}
				} else {
					if *verbose {
						fmt.Printf("  %sPrime %d: Not found%s\n", colorRed, result.Prime, colorReset)
					}
					success = false
				}
			}

			if *verbose {
				if success {
					fmt.Printf("  %sSuccess! You can cast the spell at its original level.%s\n", colorGreen, colorReset)
				} else {
					fmt.Printf("  %sFailed to find all required prime numbers.%s\n", colorRed, colorReset)
				}
				fmt.Println("-------------------------")
			} else {
				// Format status
				status := "❌"
				if success {
					status = "✅"
				}

				// Add row to table
				table.Body.Cells = append(table.Body.Cells, []*simpletable.Cell{
					{Align: simpletable.AlignLeft, Text: status},
					{Align: simpletable.AlignLeft, Text: spell.Name},
					{Align: simpletable.AlignLeft, Text: updatedRange},
					{Align: simpletable.AlignLeft, Text: diceStr},
					{Align: simpletable.AlignLeft, Text: diceStrPlus2},
					{Align: simpletable.AlignLeft, Text: empowerStr},
					{Align: simpletable.AlignLeft, Text: intensifyStr},
				})
			}
		}

		if !*verbose {
			// Set table style
			table.SetStyle(simpletable.StyleCompactLite)
			// Print column widths for debugging
			fmt.Printf("\nColumn widths:\n")
			for i, cell := range table.Header.Cells {
				fmt.Printf("Column %d (%s): %d\n", i, cell.Text, len(cell.Text))
			}
			fmt.Println(table.String())
		}
	}
}

// displayRangeInfo shows detailed information about spell ranges
func displayRangeInfo(casterLevel int) {
	fmt.Printf("Computed Range Details (Caster Level: %d):\n", casterLevel)
	for _, info := range rangeData {
		computed := info.Compute(casterLevel)
		fmt.Printf("\n%s:\n  %s\n  Computed Range: %s\n", info.Name, info.Description, computed)
	}
}

// computeDuration converts a duration in the form "X unit/level" into a computed value
func computeDuration(durationStr string, casterLevel int) string {
	if strings.Contains(durationStr, "/level") {
		parts := strings.Split(durationStr, " ")
		if len(parts) >= 2 {
			baseValue, err := strconv.Atoi(parts[0])
			if err == nil {
				unit := strings.TrimSuffix(parts[1], "/level")
				total := baseValue * casterLevel
				if total != 1 && !strings.HasSuffix(unit, "s") {
					unit += "s"
				}
				return fmt.Sprintf("%d %s", total, unit)
			}
		}
	}
	return durationStr
}
