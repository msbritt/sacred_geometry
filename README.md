# Sacred Geometry
Solves prime math problem for the Pathfinder 1E Feat Sacred Geometry

## Description
See [the Sacred Geometry page](https://aonprd.com/FeatDisplay.aspx?ItemName=Sacred%20Geometry) for details.  In short, based on the number of ranks in Knowledge (Engineering) you have, you will cast N d6 dice.  Based on the level of spell you want to cast, after applying Metamagic to it, there are 3 prime numbers (as the level goes up, so do the prime numbers).  Using the value of each d6 no more than once, you must use basic math operations (add, subtract, multiply, and divide) a subset of those dice values to generate each prime required.  If you can generate all three prime values, then you can cast the metamagic enhanced spell at its original level.

## Code
The code calculates all 3 primes in parallel, sorts the values, and alerts if you were successful or not.  To use, you must have Golang installed:

`go run sg.go 4 10`

where the first value is the metmagic level of the spell and the second is the number of points your character has in Engineering.

For additional speed, you can compile it with:
`go build -o sg sg.go`

Then run with `sg 4 10`
