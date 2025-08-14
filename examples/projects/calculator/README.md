# Calculator Service Example

This example demonstrates dynamic templates in GripMock for client streaming gRPC calls with mathematical operations.

## Features

- **SumNumbers**: Sum all numbers in the stream
- **DivideNumbers**: Divide all numbers in the stream (including division by zero test)
- **CalculateAverage**: Calculate average of all numbers
- **FindMinMax**: Find minimum and maximum values
- **ComplexCalculation**: Perform multiple operations on stream data
- **Dynamic Templates**: Use `{{.Requests}}`, `{{len .Requests}}`, `{{(index .Requests 0).value}}`, and math helpers
- **Error Handling**: Test division by zero scenarios

## Available Template Functions

- `add`: Addition
- `sub`: Subtraction  
- `mul`: Multiplication
- `div`: Division
- `len`: Length of array/string

## Project Structure

The calculator example is organized by method with separate directories:

- `sum_numbers/`: Sum all numbers in the stream
- `divide_numbers/`: Divide numbers (including division by zero test)
- `calculate_average/`: Calculate average of numbers (supports 1 or 3 messages via separate stubs)
- `find_min_max/`: Find minimum and maximum values
- `complex_calculation/`: Perform multiple operations on stream data

## How It Works

The calculator service processes client streaming requests where multiple numbers are sent in sequence. The dynamic templates use:

- `{{.Requests}}`: All received messages
- `{{len .Requests}}`: Number of messages in the stream
- `{{(index .Requests N)}}`: Access specific message by index, then its fields
- Mathematical functions to perform calculations

## Test Cases

- `case_sum_numbers.gctf`: Basic summation
- `case_divide_numbers.gctf`: Division operations
- `case_divide_by_zero.gctf`: Division by zero error case
- `case_calculate_average.gctf`: Average calculation
- `case_find_min_max.gctf`: Min/max operations
- `case_complex_calculation.gctf`: Multiple operations
- `case_large_numbers.gctf`: Large number handling
- `case_single_number.gctf`: Single number processing

## Usage

```bash
# Start the server
go run main.go examples/projects/calculator/service.proto --stub examples/projects/calculator

# Run tests
grpctestify examples/projects/calculator
```

## Dynamic Template Examples

```yaml
# Sum with real calculations
result: "{{sum (extract .Requests `value`)}}"
count: "{{len .Requests}}"

# Division with specific values
result: "{{div (index (extract .Requests `value`) 0) (index (extract .Requests `value`) 1)}}"

# Complex calculations
sum: "{{sum (extract .Requests `value`)}}"
product: "{{mul (extract .Requests `value`)}}"
average: "{{avg (extract .Requests `value`)}}"
min: "{{min (extract .Requests `value`)}}"
max: "{{max (extract .Requests `value`)}}"
```
