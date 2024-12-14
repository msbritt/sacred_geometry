package main

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strconv"
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
		return [][]string{[]string{}}
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

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: <spell_level> <engineering_ranks>")
		return
	}
	spellLevel, err1 := strconv.Atoi(os.Args[1])
	engineeringRanks, err2 := strconv.Atoi(os.Args[2])
	if err1 != nil || err2 != nil || spellLevel < 1 || spellLevel > 9 {
		fmt.Println("Please enter a valid spell level (1-9) and engineering ranks.")
		return
	}
	primes := getPrimeConstants(spellLevel)
	dice := rollDice(engineeringRanks)
	fmt.Printf("Prime constants for spell level %d: %v\n", spellLevel, primes)
	fmt.Printf("Rolling %d d6 dice: %v\n", engineeringRanks, dice)

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

	var results []Result
	for result := range resultChan {
		results = append(results, result)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Prime < results[j].Prime
	})

	success := true
	for _, result := range results {
		if result.Found {
			fmt.Printf("Combination to achieve prime %d: %s = %d\n", result.Prime, result.Expression, result.Prime)
		} else {
			fmt.Printf("No combination found to achieve prime %d\n", result.Prime)
			success = false
		}
	}

	if success {
		fmt.Println("Success: Combinations found for all prime constants.")
	} else {
		fmt.Println("Failure: Not all prime constants have combinations.")
	}
}
