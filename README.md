# Sacred Geometry

Solves prime math problem for the Pathfinder 1E Feat Sacred Geometry

## Description

See [the Sacred Geometry page](https://aonprd.com/FeatDisplay.aspx?ItemName=Sacred%20Geometry) for details.  In short, based on the number of ranks in Knowledge (Engineering) you have, you will cast N d6 dice.  Based on the level of spell you want to cast, after applying Metamagic to it, there are 3 prime numbers (as the level goes up, so do the prime numbers).  Using the value of each d6 no more than once, you must use basic math operations (add, subtract, multiply, and divide) a subset of those dice values to generate each prime required.  If you can generate all three prime values, then you can cast the metamagic enhanced spell at its original level.

## Code

The code calculates all 3 primes in parallel, sorts the values, and alerts if you were successful or not.  To use, you must have Golang installed:

### Basic Usage

```bash
go run sg.go
```

This will run the program with default settings (caster level 6, engineering 6, and process all spells in spells.csv).

### Command Line Options

You can customize the execution with command line flags:

```bash
# Enable debug mode for detailed logging
go run sg.go --debug

# Enable verbose mode for additional output
go run sg.go --verbose
```

### Configuration

The program uses the following configuration variables (defined in the code):

- `casterLevel`: Your character's caster level (default: 6)
- `engineering`: Your character's ranks in Knowledge (Engineering) (default: 6)
- `TransmuterOfKorada`: When true, Transmutation spells get +1 caster level (default: true)

### Spell Data

Spell information is read from `spells.csv` in the following format:

```csv
Name,BaseLevel,School,Range,Damage,Duration,Metamagic
Fireball,3,Evocation,long,6d6,instantaneous,empower;intensified
```

You can modify this file to add your own spells or change the metamagic feats applied.

### Compiling

For additional speed, you can compile it with:

```bash
go build -o sg sg.go
```

Then run with:

```bash
./sg
```

or with options:

```bash
./sg --debug --verbose
```

## Testing

The codebase includes comprehensive test coverage to ensure functionality and prevent regressions. Tests are written using Go's standard testing package.

### Running Tests

To run all tests:

```bash
go test
```

For more verbose output showing each test that runs:

```bash
go test -v
```

### Test Coverage

To check test coverage:

```bash
go test -cover
```

Current test coverage is approximately 59% of all statements. The core algorithmic functions have higher coverage.

For a detailed HTML report of test coverage:

```bash
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

This will open a browser window showing which lines of code are covered by tests.

### Test Structure

The test suite includes:

1. **Unit tests** for individual functions
2. **Integration tests** for spell parsing and calculations
3. **Validation tests** for the Sacred Geometry algorithm

The test suite covers the following key components:

- **Prime number constants**: Verification of prime number retrieval for different spell levels
- **Mathematical expressions**: Testing of expression evaluation with various operations
- **Spell range calculations**: Validation of range calculations for different caster levels
- **Metamagic effects**: Testing the application and stacking of different metamagic feats
- **Damage and duration parsing/formatting**: Ensuring correct parsing and formatting of spell attributes
- **CSV parsing**: Testing the reading and parsing of spell data from CSV files
- **Special features**: Testing of special features like Transmuter of Korada
- **Core algorithm**: Testing the prime number combination finder

### Adding New Tests

When adding new functionality, please also add corresponding tests. Follow the existing patterns in `sg_test.go` for consistency.

Test data files:

- `test_spells.csv`: Contains sample spells for testing the CSV parsing functionality

### Continuous Integration

For automated testing on each commit, consider setting up a CI pipeline using GitHub Actions or similar services.

### Future Test Improvements

Potential areas for expanding test coverage:

1. **Performance benchmarks**: Add benchmarks for performance-critical functions like `findCombinationToPrime`
2. **Property-based testing**: Implement property-based tests for functions like `evalExpression` to test with randomly generated inputs
3. **End-to-end tests**: Add tests that simulate the entire workflow from command-line input to output
4. **Edge cases**: Expand test coverage for edge cases and error handling
5. **Mocking**: Implement mocks for functions with side effects (like random number generation) to make tests more deterministic
