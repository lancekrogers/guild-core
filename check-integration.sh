#!/bin/bash

echo "=== GUILD INTEGRATION TEST STATUS ==="
echo "Date: $(date)"
echo ""

# Counter
pass=0
fail=0

# Find all integration test directories (with actual test files)
for dir in $(find integration -type d -maxdepth 2 | grep -v "^integration$" | sort); do
    # Skip if no test files
    if ! ls $dir/*_test.go >/dev/null 2>&1; then
        continue
    fi
    testname=$(basename $dir)
    parent=$(basename $(dirname $dir))
    
    # Build display name
    if [ "$parent" = "integration" ]; then
        display=$testname
    else
        display="$parent/$testname"
    fi
    
    # Run test with timeout
    printf "%-35s " "$display:"
    
    if timeout 30s go test -tags integration ./$dir -count=1 >/dev/null 2>&1; then
        echo "✅ PASS"
        ((pass++))
    else
        # Check if it's a build failure
        if go test -tags integration -c ./$dir >/dev/null 2>&1; then
            echo "❌ FAIL (test failure)"
        else
            echo "❌ FAIL (build failure)"
        fi
        ((fail++))
    fi
done

echo ""
echo "=== SUMMARY ==="
echo "Passing: $pass"
echo "Failing: $fail"
echo "Total: $((pass + fail))"
echo "Success Rate: $(( pass * 100 / (pass + fail) ))%"