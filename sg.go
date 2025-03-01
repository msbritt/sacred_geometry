package main

import (
	"bufio"
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
			School:     record[2],
			Range:      record[3],
			DamageRoll: parseDamage(record[4], record[0]),
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

		// Use the global caster level instead of setting it to the spell level
		// casterLevel = spellLevel  // This line is commented out

		// Apply Transmuter of Korada bonus if applicable
		spellCasterLevel := casterLevel // Use a local variable for this spell's caster level
		if TransmuterOfKorada && strings.ToLower(spell.School) == "transmutation" {
			spellCasterLevel += 1
			if debugMode {
				fmt.Printf("Debug: Applied Transmuter of Korada bonus (+1 caster level) to %s\n", spell.Name)
			}
		}

		// Initialize range calculator
		ranges := NewRangeCalculator(spellCasterLevel)

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

		// Add Transmuter of Korada information if applicable
		if TransmuterOfKorada && strings.ToLower(spell.School) == "transmutation" {
			metamagicStr += fmt.Sprintf("; Transmuter of Korada: +1 caster level (CL %d)", spellCasterLevel)
		}

		fmt.Printf("\nCalculating %s: %s\n", spellCopy.Name, metamagicStr)

		if debugMode {
			fmt.Printf("Debug: Before applyMetamagicEffects - MaxDice: %d\n", spellCopy.DamageRoll.MaxDice)
		}
		applyMetamagicEffects(&spellCopy)
		if debugMode {
			fmt.Printf("Debug: After applyMetamagicEffects - MaxDice: %d\n", spellCopy.DamageRoll.MaxDice)
		}

		primes := getPrimeConstants(spellLevel)
		dice := rollDice(engineering)

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
			fmt.Printf("\n%sSuccess:%s Sacred Geometry succeeded for %s!\n", colorGreen, colorReset, spellCopy.Name)

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
				baseDamage := formatDamage(spell.DamageRoll, spellCasterLevel, spell.Name)

				// Special handling for Magic Missile display
				if spell.Name == "Magic Missile" {
					// Calculate number of missiles based on caster level
					projectiles := 1 + (spellCasterLevel-1)/2
					if projectiles > 5 {
						projectiles = 5 // Maximum of 5 missiles
					}
					fmt.Printf("Base Damage: %s\n", baseDamage)
					fmt.Printf("Number of Missiles: %d (1 at level 1, +1 per 2 levels, max 5)\n", projectiles)
				} else if spell.DamageRoll.PerLevel {
					actualDice := spell.DamageRoll.NumDice * spellCasterLevel
					if actualDice > spell.DamageRoll.MaxDice {
						actualDice = spell.DamageRoll.MaxDice
					}
					fmt.Printf("Base Damage: %s (%dd%d, max %d dice)\n",
						baseDamage, actualDice, spell.DamageRoll.DiceType, spell.DamageRoll.MaxDice)
				} else {
					fmt.Printf("Base Damage: %s\n", baseDamage)
				}

				// Show Intensified damage if present
				for _, metamagic := range spellCopy.MetamagicFeats {
					if strings.ToLower(metamagic) == "intensified" {
						// Calculate actual dice after intensified
						intensifiedDamage := formatDamage(spellCopy.DamageRoll, spellCasterLevel, spell.Name)
						actualDice := spellCopy.DamageRoll.NumDice * spellCasterLevel
						if spellCopy.DamageRoll.PerLevel && actualDice > spellCopy.DamageRoll.MaxDice {
							actualDice = spellCopy.DamageRoll.MaxDice
						}
						fmt.Printf("Intensified Damage: %s (%d dice, max dice increased to %d)\n",
							intensifiedDamage, actualDice, spellCopy.DamageRoll.MaxDice)
						break
					}
				}

				// Show Empowered damage if present
				for _, metamagic := range spellCopy.MetamagicFeats {
					if strings.ToLower(metamagic) == "empower" {
						empoweredDamage := formatDamage(spellCopy.DamageRoll, spellCasterLevel, spell.Name)

						// Special handling for Magic Missile with Empower
						if spell.Name == "Magic Missile" {
							fmt.Printf("Empowered Damage: %s (damage ×1.5)\n", empoweredDamage)
						} else {
							fmt.Printf("Empowered Damage: %s (×1.5)\n", empoweredDamage)
						}
						break
					}
				}
			}

			if spellCopy.Duration.Value > 0 {
				fmt.Printf("Duration: %s\n", formatDuration(spellCopy.Duration, spellCasterLevel))
			}
		} else {
			fmt.Printf("\n%sFailure:%s Sacred Geometry failed for %s\n", colorRed, colorReset, spellCopy.Name)
		}
	}
}
