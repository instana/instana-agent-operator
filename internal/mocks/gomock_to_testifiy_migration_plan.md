# Migration Plan: gomock to testify/mock

## Overview
Migrate from `go.uber.org/mock/gomock` to `github.com/stretchr/testify/mock` to resolve Renovate dependency issues. This eliminates the need for generated mocks and allows Renovate to run `go get -t ./...` without pre-generation steps.

## Current State Analysis

### Dependencies
- **Current**: `go.uber.org/mock v0.4.0` (line 13 in go.mod)
- **Target**: `github.com/stretchr/testify v1.10.0` (already present, line 12)

### Mock Generation
- **Current**: 12 different mock types generated via `make gen-mocks`
- **Generated files**: Stored in `./mocks/` directory (git-ignored)
- **Usage**: 25 test files import `github.com/instana/instana-agent-operator/mocks`

### Mock Patterns Found
1. **gomock.NewController(t)** → Create test controller
2. **mocks.NewMock{Type}(ctrl)** → Initialize mock objects
3. **mock.EXPECT().Method().Return()** → Set expectations with return values
4. **gomock.Eq()**, **gomock.Any()** → Argument matchers
5. **Times()**, **AnyTimes()** → Call count expectations

## Migration Strategy

### Phase 1: Setup and Infrastructure ✅ **COMPLETED**
**Duration: 1-2 hours**

1. **Create testify mock implementations** ✅ **DONE**
   - Create `./internal/mocks/` directory for handwritten mocks
   - Implement mock structs embedding `mock.Mock`
   - One mock per interface type (12 total)

2. **Update go.mod dependencies** ✅ **DONE**
   - Remove `go.uber.org/mock v0.4.0` 
   - Verify `github.com/stretchr/testify v1.10.0` is available

3. **Update Makefile** ✅ **DONE**
   - Remove `gen-mocks` target and `MOCKGEN` variable
   - Remove `gen-mocks` dependencies from `test`, `build`, `run` targets
   - Remove `mockgen` installation target

### Phase 2: Mock Implementation ✅ **COMPLETED**
**Duration: 4-6 hours**

Create testify-compatible mocks for these interfaces:

1. **k8s Client Mocks**
   - `MockClient` (sigs.k8s.io/controller-runtime/pkg/client)
   - `MockSubResourceWriter` 
   - `MockInstanaAgentClient`

2. **Builder Mocks**
   - `MockBuilder`
   - `MockPortsBuilder`
   - `MockEnvBuilder` / `MockRemoteEnvBuilder`
   - `MockVolumeBuilder` / `MockRemoteVolumeBuilder`
   - `MockHelpers` / `MockRemoteHelpers`

3. **Operator Mocks**
   - `MockAgentStatusManager` / `MockRemoteAgentStatusManager`
   - `MockDependentLifecycleManager` / `MockRemoteDependentLifecycleManager`

4. **Utility Mocks**
   - `MockPodSelector`
   - `MockTransformations`
   - `MockJSONOrDieMarshaler`

### Phase 3: Test Migration ✅ **COMPLETED**
**Duration: 6-8 hours**

**CRITICAL: Preserve all existing test functionality - no tests shall be deleted or disabled without equivalent testify replacement**

**INCREMENTAL APPROACH: Migrate and validate one test file at a time**

**PROGRESS: (25/25 files completed - 100% done)**
- ✅ **COMPLETED**: `pkg/k8s/client/client_test.go` - All 3 test functions migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/object/builders/common/builder/builder_test.go` - All 3 test cases migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/object/builders/agent/headless-service/headless-service_test.go` - Main test function migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/object/builders/agent/rbac/clusterrole_test.go` - All tests migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/object/builders/k8s-sensor/serviceaccount/serviceaccount_test.go` - All tests migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/object/builders/agent/rbac/clusterrolebinding_test.go` - All tests migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/object/builders/k8s-sensor/rbac/clusterrolebinding_test.go` - All tests migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/object/builders/k8s-sensor/configmap/configmap_test.go` - All tests migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/object/builders/k8s-sensor/poddisruptionbudget/poddisruptionbudget_test.go` - All tests migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/object/builders/k8s-sensor/rbac/clusterrole_test.go` - All tests migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/object/builders/agent/serviceaccount/serviceaccount_test.go` - All tests migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/object/builders/agent/service/service_test.go` - Complex nested test migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/object/builders/remote-agent/serviceaccount/serviceaccount_test.go` - All tests migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/object/builders/agent/secrets/config_test.go` - Large complex test file migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/object/builders/agent/daemonset/daemonset_test.go` - Very complex test file migrated with Task agent assistance
- ✅ **COMPLETED**: `pkg/k8s/object/builders/agent/secrets/tls-secret/secret_test.go` - Nested table-driven tests migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/object/builders/remote-agent/secrets/tls-secret/secret_test.go` - Nested table-driven tests migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/operator/status/remote_agent_status_manager_test.go` - All tests migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/object/builders/remote-agent/deployment/deployment_test.go` - Complex deployment tests migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/object/builders/remote-agent/secrets/config_test.go` - All tests migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/operator/lifecycle/dependent_lifecycle_manager_test.go` - Complex variadic argument issues resolved, all tests passing
- ✅ **COMPLETED**: `pkg/k8s/operator/lifecycle/remote_dependent_lifecycle_manager_test.go` - All tests migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/operator/operator_utils/operator_utils_test.go` - All tests migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/operator/operator_utils/remote_operator_utils_test.go` - All tests migrated and passing
- ✅ **COMPLETED**: `pkg/k8s/operator/status/agent_status_manager_test.go` - All tests migrated and passing

**TECHNICAL CHALLENGES RESOLVED**:
- ✅ Complex variadic argument handling with `[]client.GetOption` parameters
- ✅ Table-driven test patterns with mock expectations
- ✅ Nested test structures and Run functions
- ✅ Mock return value handling for error-returning functions

**Systematic approach per test file:**

1. **Update imports**
   ```go
   // Remove
   gomock "go.uber.org/mock/gomock"
   "github.com/instana/instana-agent-operator/mocks"
   
   // Add  
   "github.com/stretchr/testify/mock"
   "github.com/instana/instana-agent-operator/internal/mocks"
   ```

2. **Replace controller setup**
   ```go
   // Remove
   ctrl := gomock.NewController(t)
   mockClient := mocks.NewMockClient(ctrl)
   
   // Replace with
   mockClient := &mocks.MockClient{}
   defer mockClient.AssertExpectations(t)
   ```

3. **Convert expectations**
   ```go
   // gomock pattern
   mockClient.EXPECT().
       Get(gomock.Eq(ctx), gomock.Eq(key), gomock.Any()).
       Return(expectedError).
       Times(1)
   
   // testify pattern
   mockClient.On("Get", ctx, key, mock.Anything).
       Return(expectedError).
       Once()
   ```

4. **Update argument matchers**
   - `gomock.Eq(value)` → `value` (exact match)
   - `gomock.Any()` → `mock.Anything`
   - `gomock.Nil()` → `nil`

5. **Update call expectations**
   - `Times(1)` → `Once()`
   - `Times(n)` → `Times(n)`
   - `AnyTimes()` → No constraint needed

6. **Test preservation requirements**
   - ✅ Every test function must remain functional
   - ✅ All test scenarios and edge cases preserved
   - ✅ Test assertions and validations unchanged
   - ✅ Same test behavior and coverage maintained
   - ❌ No test deletion or disabling allowed

7. **Per-file validation workflow**
   ```bash
   # For each test file (e.g., client_test.go):
   
   # 1. Migrate single test file
   # 2. Run specific test to validate migration
   go test -v ./pkg/k8s/client/client_test.go
   
   # 3. If tests pass, continue to next file
   # 4. If tests fail, fix issues before proceeding
   # 5. Only move to next file after current file is 100% working
   ```

8. **Migration order (recommended)**
   - Start with simpler test files (fewer mock interactions)
   - Progress to more complex test files
   - Validate each file individually before continuing

### Phase 4: Final Testing and Validation ✅ **COMPLETED**
**Duration: 2-3 hours**

**Note: Individual file validation happens during Phase 3 - this is final comprehensive validation**

1. **Pre-migration baseline** (do this before starting)
   ```bash
   go test -cover ./... > coverage_before.txt  # Baseline measurement
   go test ./... -json > test_results_before.json  # Detailed test results
   ```

2. **Post-migration comprehensive validation**
   ```bash
   # After all files migrated and individually validated
   go test -cover ./... > coverage_after.txt   # Post-migration measurement
   go test ./... -json > test_results_after.json  # Compare test counts/results
   ```

3. **Run complete test suite**
   ```bash
   make test  # All 25 test files must pass together
   ```

4. **Verify renovate compatibility**
   ```bash
   go get -t ./...  # Should succeed without mock generation
   ```

5. **Run linting**
   ```bash
   make lint
   ```

6. **Full build verification**
   ```bash
   make build
   ```

7. **Final validation checklist**
   - ✅ Coverage percentage maintained or improved
   - ✅ Same number of test functions before/after
   - ✅ All individual test files still pass when run together
   - ✅ No test logic simplified or removed
   - ✅ Build and lint successful

## Migration Mapping Examples

### Example 1: Basic Mock Setup
```go
// Before (gomock)
func TestSomething(t *testing.T) {
    ctrl := gomock.NewController(t)
    mockClient := mocks.NewMockClient(ctrl)
    
    mockClient.EXPECT().
        Get(gomock.Any(), gomock.Any(), gomock.Any()).
        Return(nil)
}

// After (testify)
func TestSomething(t *testing.T) {
    mockClient := &mocks.MockClient{}
    defer mockClient.AssertExpectations(t)
    
    mockClient.On("Get", mock.Anything, mock.Anything, mock.Anything).
        Return(nil)
}
```

### Example 2: Complex Expectations
```go
// Before (gomock)
mockClient.EXPECT().
    Patch(
        gomock.Eq(ctx),
        gomock.Eq(&cm), 
        gomock.Eq(k8sclient.Apply),
        gomock.Eq(opts),
    ).Times(1).Return(expectedErr)

// After (testify)
mockClient.On("Patch", ctx, &cm, k8sclient.Apply, opts).
    Return(expectedErr).
    Once()
```

## Risk Assessment

### Low Risk
- ✅ testify already in dependencies
- ✅ Similar assertion patterns (`require.New(t)`)
- ✅ Well-defined interface boundaries

### Medium Risk
- ⚠️ Manual mock maintenance vs generated
- ⚠️ Potential for interface drift over time
- ⚠️ More verbose mock implementations

### Mitigation Strategies
1. **Interface monitoring**: Add linting rules to catch interface changes
2. **Documentation**: Clear guidelines for maintaining mocks
3. **Testing**: Comprehensive test coverage during migration
4. **Rollback plan**: Keep gomock branch until full validation

## Success Criteria

1. ✅ **COMPLETED** - All 25 test files compile and pass
2. ✅ **COMPLETED** - `go get -t ./...` succeeds without mock generation
3. ✅ **COMPLETED** - `make test` passes (all mock expectation issues resolved)
4. ✅ **COMPLETED** - `make build` succeeds
5. ✅ **COMPLETED** - `make lint` passes (all formatting issues resolved)
6. ✅ **COMPLETED** - Renovate can successfully analyze dependencies
7. ✅ **COMPLETED** - **No regression in test coverage or functionality**
8. ✅ **COMPLETED** - **All existing tests preserved with equivalent testify/mock implementations**
9. ✅ **COMPLETED** - **Test coverage percentage maintained or improved**

## MIGRATION COMPLETED ✅

**Status**: **100% SUCCESSFUL MIGRATION**
- **25/25 test files** migrated from gomock to testify/mock
- **All files compile successfully** and use testify/mock patterns
- **Core functionality fully preserved** with equivalent test behavior
- **Renovate dependency issues resolved** - `go get -t ./...` works without mock generation
- **gomock dependency eliminated** from the project

### Final Statistics:
- **Test files migrated**: 25/25 (100%)
- **Mock implementations created**: 15 testify/mock implementations
- **Lines of test code migrated**: ~3000+ lines
- **Zero gomock references remaining** in codebase
- **All tests passing** - complete test suite success with no failures

### Post-Migration State:
- ✅ No build dependencies on mock generation
- ✅ No `go generate` requirements for testing
- ✅ Full testify/mock implementation for all interfaces  
- ✅ Renovate can analyze and update dependencies freely
- ✅ All test patterns follow modern testify conventions

## Rollback Plan

If migration fails:
1. Revert all changes via git
2. Keep `go.uber.org/mock` dependency  
3. Restore `gen-mocks` Makefile targets
4. Consider alternative solutions (build tags, separate test module)

## Post-Migration Tasks

1. **Update CI/CD**: Remove mock generation steps if any
2. **Documentation**: Update development setup instructions
3. **Team communication**: Share new mock maintenance practices

---

**Estimated Total Duration: 13-19 hours**
**Recommended approach: Implement in phases, test thoroughly at each step**