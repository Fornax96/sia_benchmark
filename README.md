# Sia benchmarking tool

This tool can be used for benchmarking Sia's upload performance and storage capacity.

## Build instructions

```bash
# Clone the repo into your GOPATH
go get -u github.com/Fornax96/sia_benchmark

# Navigate to the repo
cd $GOPATH/src/Fornax96/sia_benchmark

# Build the binary
go build main.go

# The program will be called main, move it to the place where you'll be using it from
mv main ~/benchmark
```

## Usage instructions

Running the program will generate a default config file called `benchmark.toml`
in your present working directory. You can tweak the values in there if you
want. If you run the program again it will start the test with the configured
parameters.

The benchmark tool will not set the Sia allowance for you (yet). So you need to
do that yourself with this command before starting the test:

```bash
siac renter setallowance --amount 10KS --hosts 50 --period 12w --renew-window 2w
```
(parameters are tweakable of course)

The benchmark tool will generate files of your configured size in a directory
called `upload_queue` (configurable too). If you end the test some files might
be left over in this directory, you have to empty the directory before starting
a new test.

During the test the tool will write the results to a CSV spreadsheet called `metrics.csv` in the present working directory. These metrics can be interpreted by Hakkane's test parser in order to use them for displaying graphs (https://github.com/hakkane84/sia-test-parser).

## Results

The results of the tests which are run by the STAC (Sia Test App Community) are
available from Siastats.info (https://siastats.info/benchmarking).
