# Test Failure Analysis

## Summary
6 packages have failing tests with 3 main categories of issues:

1. **Build failures** (cmd/guild, internal/chat, pkg/campaign) - due to undefined methods/fields
2. **Test assertion failures** (pkg/prompts/standard/evaluation, tools/code, tools/edit) - tests expecting different output/behavior
3. **External command failures** (tools/code) - go list command failing

## Detailed Analysis by Package

### 1. cmd/guild - BUILD FAILED
**Root Cause**: pkg/grpc package compilation errors
- `campaign.TotalObjectives` undefined
- `campaign.CompletedObjectives` undefined  
- `campaignMgr.AddObjective` undefined
- `campaignMgr.RemoveObjective` undefined
- `campaign.Objectives` undefined

**Fix Required**: Update pkg/grpc to match current campaign package API

### 2. internal/chat - BUILD FAILED
**Root Cause**: Same pkg/grpc compilation errors as cmd/guild
**Fix Required**: Same as above

### 3. pkg/campaign - BUILD FAILED
**Root Cause**: Missing method in test
- `repo.GetByObjectiveID` undefined in campaign_test.go:277

**Fix Required**: Either implement GetByObjectiveID method or update test

### 4. pkg/prompts/standard/evaluation - 1 TEST FAILED
**Failing Test**: TestPromptEvaluator
- Error: "Expected at least some tests to pass" at evaluator_test.go:80
- All other tests in package pass

**Fix Required**: Review test expectations or implementation

### 5. tools/code - 11 TESTS FAILED
**Failing Tests**:
1. `TestDependenciesTool_Execute_GoMod` - go list command fails
2. `TestDependenciesTool_Execute_UnknownProject` - expects nil but gets error
3. `TestDependenciesTool_Execute_WithFilters` - go list command fails
4. `TestDependenciesTool_Execute_Outdated` - go list command fails
5. `TestDependenciesTool_Execute_AllFormats` (3 subtests) - go list command fails
6. `TestMetricsTool_Execute_OnlyComplexity` - output format mismatch
7. `TestMetricsTool_Execute_OnlyLOC` - output format mismatch
8. `TestSearchReplaceTool_Execute_InvalidRegex` - expects nil but gets error
9. `TestSearchReplaceTool_Execute_MaxResults` - expects "3 matches" but gets "5 matches"

**Fix Categories**:
- Go environment issues (go list failures)
- Output format expectations
- Error handling expectations

### 6. tools/edit - 8 TESTS FAILED
**Failing Tests**:
1. `TestCursorPositionTool_Execute_NonexistentMark` - expects nil but gets error
2. `TestCursorPositionTool_Execute_NoOperation` - expects nil but gets error
3. `TestMultiFileRefactorTool_Execute_NamingConflict` - missing "Conflicts" in output
4. `TestMultiFileRefactorTool_Execute_InvalidType` - expects nil but gets error
5. `TestMultiFileRefactorTool_Execute_RenameWithoutNewName` - expects nil but gets error
6. `TestMultiFileRefactorTool_Execute_ExtractWithoutLines` - expects nil but gets error
7. `TestMultiFileRefactorTool_Execute_MoveWithoutDestination` - expects nil but gets error
8. `TestMultiFileRefactorTool_Execute_ExtractOutOfBounds` - expects nil but gets error

**Fix Categories**:
- Error handling expectations (most tests expect nil but get proper error objects)
- Output format expectations (naming conflict test)

## Priority Order for Fixes

1. **High Priority - Build Failures** (blocks all testing):
   - Fix pkg/grpc to match campaign package API
   - Fix pkg/campaign test method call

2. **Medium Priority - Test Logic**:
   - Fix tools/edit error handling expectations
   - Fix tools/code output format and error expectations
   - Fix pkg/prompts/standard/evaluation test logic

3. **Low Priority - Environment Issues**:
   - Investigate go list command failures in tools/code tests

## Next Steps
1. Start with fixing the campaign API mismatch in pkg/grpc
2. Update campaign repository test to use correct method
3. Review and update test expectations for error handling
4. Fix output format expectations in metrics and search tools