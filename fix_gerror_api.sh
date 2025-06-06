#!/bin/bash

# Fix gerror API usage throughout the codebase

echo "Fixing gerror API usage..."

# Fix undefined error constants
find . -name "*.go" -type f | xargs sed -i '' \
  -e 's/gerror\.InvalidArgument/gerror.ErrCodeInvalidInput/g' \
  -e 's/gerror\.Internal/gerror.ErrCodeInternal/g' \
  -e 's/gerror\.NotFound/gerror.ErrCodeNotFound/g' \
  -e 's/gerror\.AlreadyExists/gerror.ErrCodeAlreadyExists/g' \
  -e 's/gerror\.Validation/gerror.ErrCodeValidation/g' \
  -e 's/gerror\.Storage/gerror.ErrCodeStorage/g' \
  -e 's/gerror\.InvalidFormat/gerror.ErrCodeInvalidFormat/g' \
  -e 's/gerror\.OutOfRange/gerror.ErrCodeOutOfRange/g' \
  -e 's/gerror\.Timeout/gerror.ErrCodeTimeout/g' \
  -e 's/gerror\.Cancelled/gerror.ErrCodeCancelled/g' \
  -e 's/gerror\.Connection/gerror.ErrCodeConnection/g' \
  -e 's/gerror\.Provider/gerror.ErrCodeProvider/g' \
  -e 's/gerror\.Agent/gerror.ErrCodeAgent/g' \
  -e 's/gerror\.Orchestration/gerror.ErrCodeOrchestration/g'

# Fix gerror.Wrap calls with old syntax (4-5 arguments to 3 arguments)
# This is more complex and would need manual fixing for proper error messages

# Fix gerror.New calls with old syntax (4-5 arguments to 3 arguments)
# This is also complex and needs manual fixing

echo "Basic constant replacements done."
echo "Note: You'll need to manually fix gerror.New() and gerror.Wrap() calls"
echo "to use the new API signature:"
echo "  - gerror.New(code, message, cause)"
echo "  - gerror.Wrap(err, code, message)"
echo "And chain .WithComponent() and .WithOperation() as needed"