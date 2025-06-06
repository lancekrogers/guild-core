#!/usr/bin/env python3

import re
import sys
import os

def fix_gerror_calls(filename):
    """Fix gerror.New and gerror.Wrap calls to match new API."""
    
    with open(filename, 'r') as f:
        content = f.read()
    
    original_content = content
    
    # Fix gerror.New with 4 or 5 arguments
    # Old: gerror.New(code, component, operation, message, ...)
    # New: gerror.New(code, message, nil).WithComponent(component).WithOperation(operation)
    pattern1 = r'gerror\.New\((gerror\.\w+),\s*"([^"]+)",\s*"([^"]+)",\s*"([^"]+)"(?:,\s*(.+?))?\)'
    
    def replace_new(match):
        code = match.group(1)
        component = match.group(2)
        operation = match.group(3)
        message = match.group(4)
        extra = match.group(5)
        
        if extra and extra.strip():
            # If there are format arguments, use Newf
            return f'gerror.Newf({code}, {message}, {extra}).\n\t\t\tWithComponent("{component}").\n\t\t\tWithOperation("{operation}")'
        else:
            return f'gerror.New({code}, "{message}", nil).\n\t\t\tWithComponent("{component}").\n\t\t\tWithOperation("{operation}")'
    
    content = re.sub(pattern1, replace_new, content)
    
    # Fix gerror.Wrap with 5 arguments
    # Old: gerror.Wrap(err, code, component, operation, message)
    # New: gerror.Wrap(err, code, message).WithComponent(component).WithOperation(operation)
    pattern2 = r'gerror\.Wrap\(([^,]+),\s*(gerror\.\w+),\s*"([^"]+)",\s*"([^"]+)",\s*"([^"]+)"(?:,\s*(.+?))?\)'
    
    def replace_wrap(match):
        err = match.group(1)
        code = match.group(2)
        component = match.group(3)
        operation = match.group(4)
        message = match.group(5)
        extra = match.group(6)
        
        if extra and extra.strip():
            # If there are format arguments, use Wrapf
            return f'gerror.Wrapf({err}, {code}, "{message}", {extra}).\n\t\t\tWithComponent("{component}").\n\t\t\tWithOperation("{operation}")'
        else:
            return f'gerror.Wrap({err}, {code}, "{message}").\n\t\t\tWithComponent("{component}").\n\t\t\tWithOperation("{operation}")'
    
    content = re.sub(pattern2, replace_wrap, content)
    
    # Fix missing fmt import if needed
    if 'undefined: fmt' in content or 'fmt.' in content:
        if 'import (' in content and '"fmt"' not in content:
            content = re.sub(r'(import \()', r'\1\n\t"fmt"', content, 1)
    
    # Only write if changed
    if content != original_content:
        with open(filename, 'w') as f:
            f.write(content)
        print(f"Fixed: {filename}")
        return True
    return False

def main():
    # Files with gerror issues
    files_to_fix = [
        "tools/tool.go",
        "pkg/prompts/standard/metadata.go",
        "pkg/workspace/git_integration.go",
        "pkg/storage/board_repository.go",
        "pkg/memory/vector/chromem.go",
        "pkg/mcp/transport/memory.go",
    ]
    
    fixed_count = 0
    for file in files_to_fix:
        if os.path.exists(file):
            if fix_gerror_calls(file):
                fixed_count += 1
    
    print(f"\nFixed {fixed_count} files")

if __name__ == "__main__":
    main()