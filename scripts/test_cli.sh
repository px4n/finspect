#!/bin/bash

# Test script for finspect CLI

echo "Testing finspect CLI..."
echo "====================="

# Build the project
echo "1. Building..."
make build
if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

# Create test directory
TEST_DIR="/tmp/finspect-test-$$"
mkdir -p "$TEST_DIR/data"
echo "Test content 1" >"$TEST_DIR/data/file1.txt"
echo "Test content 2" >"$TEST_DIR/data/file2.txt"
mkdir -p "$TEST_DIR/data/subdir"
echo "Test content 3" >"$TEST_DIR/data/subdir/file3.txt"

echo ""
echo "2. Testing version command..."
./bin/finspect version

echo ""
echo "3. Testing mount command..."
./bin/finspect mount local "$TEST_DIR/data" /test

echo ""
echo "4. Testing ls command..."
./bin/finspect ls /test

echo ""
echo "5. Testing ls -l command..."
./bin/finspect ls -l /test

echo ""
echo "6. Testing stat command..."
./bin/finspect stat /test/file1.txt

echo ""
echo "7. Testing cp command..."
./bin/finspect cp /test/file1.txt /test/file1_copy.txt
./bin/finspect ls /test

echo ""
echo "8. Testing mv command..."
./bin/finspect mv /test/file1_copy.txt /test/file1_renamed.txt
./bin/finspect ls /test

echo ""
echo "9. Testing rm command..."
./bin/finspect rm /test/file1_renamed.txt
./bin/finspect ls /test

echo ""
echo "10. Cleanup..."
rm -rf "$TEST_DIR"

echo ""
echo "All tests completed!"
