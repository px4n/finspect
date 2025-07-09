#!/bin/bash

echo "Removing empty base_adaptor.go file..."
if [ -f adaptors/cloud/base_adaptor.go ]; then
    rm adaptors/cloud/base_adaptor.go
    echo "Removed adaptors/cloud/base_adaptor.go"
fi

echo "Running go vet..."
go vet ./...

echo "Running go fmt..."
go fmt ./...

echo "Done!"