// Code generated by go-mockgen 1.3.6; DO NOT EDIT.
//
// This file was generated by running `sg generate` (or `go-mockgen`) at the root of
// this repository. To add additional mocks to this or another package, add a new entry
// to the mockgen.yaml file in the root of this repository.

package policies

import (
	"context"
	"sync"
	"time"

	gitdomain "github.com/sourcegraph/sourcegraph/internal/gitserver/gitdomain"
)

// MockGitserverClient is a mock implementation of the GitserverClient
// interface (from the package
// github.com/sourcegraph/sourcegraph/internal/codeintel/policies/enterprise)
// used for unit testing.
type MockGitserverClient struct {
	// CommitDateFunc is an instance of a mock function object controlling
	// the behavior of the method CommitDate.
	CommitDateFunc *GitserverClientCommitDateFunc
	// CommitsUniqueToBranchFunc is an instance of a mock function object
	// controlling the behavior of the method CommitsUniqueToBranch.
	CommitsUniqueToBranchFunc *GitserverClientCommitsUniqueToBranchFunc
	// RefDescriptionsFunc is an instance of a mock function object
	// controlling the behavior of the method RefDescriptions.
	RefDescriptionsFunc *GitserverClientRefDescriptionsFunc
}

// NewMockGitserverClient creates a new mock of the GitserverClient
// interface. All methods return zero values for all results, unless
// overwritten.
func NewMockGitserverClient() *MockGitserverClient {
	return &MockGitserverClient{
		CommitDateFunc: &GitserverClientCommitDateFunc{
			defaultHook: func(context.Context, int, string) (r0 string, r1 time.Time, r2 bool, r3 error) {
				return
			},
		},
		CommitsUniqueToBranchFunc: &GitserverClientCommitsUniqueToBranchFunc{
			defaultHook: func(context.Context, int, string, bool, *time.Time) (r0 map[string]time.Time, r1 error) {
				return
			},
		},
		RefDescriptionsFunc: &GitserverClientRefDescriptionsFunc{
			defaultHook: func(context.Context, int, ...string) (r0 map[string][]gitdomain.RefDescription, r1 error) {
				return
			},
		},
	}
}

// NewStrictMockGitserverClient creates a new mock of the GitserverClient
// interface. All methods panic on invocation, unless overwritten.
func NewStrictMockGitserverClient() *MockGitserverClient {
	return &MockGitserverClient{
		CommitDateFunc: &GitserverClientCommitDateFunc{
			defaultHook: func(context.Context, int, string) (string, time.Time, bool, error) {
				panic("unexpected invocation of MockGitserverClient.CommitDate")
			},
		},
		CommitsUniqueToBranchFunc: &GitserverClientCommitsUniqueToBranchFunc{
			defaultHook: func(context.Context, int, string, bool, *time.Time) (map[string]time.Time, error) {
				panic("unexpected invocation of MockGitserverClient.CommitsUniqueToBranch")
			},
		},
		RefDescriptionsFunc: &GitserverClientRefDescriptionsFunc{
			defaultHook: func(context.Context, int, ...string) (map[string][]gitdomain.RefDescription, error) {
				panic("unexpected invocation of MockGitserverClient.RefDescriptions")
			},
		},
	}
}

// NewMockGitserverClientFrom creates a new mock of the MockGitserverClient
// interface. All methods delegate to the given implementation, unless
// overwritten.
func NewMockGitserverClientFrom(i GitserverClient) *MockGitserverClient {
	return &MockGitserverClient{
		CommitDateFunc: &GitserverClientCommitDateFunc{
			defaultHook: i.CommitDate,
		},
		CommitsUniqueToBranchFunc: &GitserverClientCommitsUniqueToBranchFunc{
			defaultHook: i.CommitsUniqueToBranch,
		},
		RefDescriptionsFunc: &GitserverClientRefDescriptionsFunc{
			defaultHook: i.RefDescriptions,
		},
	}
}

// GitserverClientCommitDateFunc describes the behavior when the CommitDate
// method of the parent MockGitserverClient instance is invoked.
type GitserverClientCommitDateFunc struct {
	defaultHook func(context.Context, int, string) (string, time.Time, bool, error)
	hooks       []func(context.Context, int, string) (string, time.Time, bool, error)
	history     []GitserverClientCommitDateFuncCall
	mutex       sync.Mutex
}

// CommitDate delegates to the next hook function in the queue and stores
// the parameter and result values of this invocation.
func (m *MockGitserverClient) CommitDate(v0 context.Context, v1 int, v2 string) (string, time.Time, bool, error) {
	r0, r1, r2, r3 := m.CommitDateFunc.nextHook()(v0, v1, v2)
	m.CommitDateFunc.appendCall(GitserverClientCommitDateFuncCall{v0, v1, v2, r0, r1, r2, r3})
	return r0, r1, r2, r3
}

// SetDefaultHook sets function that is called when the CommitDate method of
// the parent MockGitserverClient instance is invoked and the hook queue is
// empty.
func (f *GitserverClientCommitDateFunc) SetDefaultHook(hook func(context.Context, int, string) (string, time.Time, bool, error)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// CommitDate method of the parent MockGitserverClient instance invokes the
// hook at the front of the queue and discards it. After the queue is empty,
// the default hook function is invoked for any future action.
func (f *GitserverClientCommitDateFunc) PushHook(hook func(context.Context, int, string) (string, time.Time, bool, error)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultHook with a function that returns the
// given values.
func (f *GitserverClientCommitDateFunc) SetDefaultReturn(r0 string, r1 time.Time, r2 bool, r3 error) {
	f.SetDefaultHook(func(context.Context, int, string) (string, time.Time, bool, error) {
		return r0, r1, r2, r3
	})
}

// PushReturn calls PushHook with a function that returns the given values.
func (f *GitserverClientCommitDateFunc) PushReturn(r0 string, r1 time.Time, r2 bool, r3 error) {
	f.PushHook(func(context.Context, int, string) (string, time.Time, bool, error) {
		return r0, r1, r2, r3
	})
}

func (f *GitserverClientCommitDateFunc) nextHook() func(context.Context, int, string) (string, time.Time, bool, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *GitserverClientCommitDateFunc) appendCall(r0 GitserverClientCommitDateFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of GitserverClientCommitDateFuncCall objects
// describing the invocations of this function.
func (f *GitserverClientCommitDateFunc) History() []GitserverClientCommitDateFuncCall {
	f.mutex.Lock()
	history := make([]GitserverClientCommitDateFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// GitserverClientCommitDateFuncCall is an object that describes an
// invocation of method CommitDate on an instance of MockGitserverClient.
type GitserverClientCommitDateFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 context.Context
	// Arg1 is the value of the 2nd argument passed to this method
	// invocation.
	Arg1 int
	// Arg2 is the value of the 3rd argument passed to this method
	// invocation.
	Arg2 string
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 string
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 time.Time
	// Result2 is the value of the 3rd result returned from this method
	// invocation.
	Result2 bool
	// Result3 is the value of the 4th result returned from this method
	// invocation.
	Result3 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c GitserverClientCommitDateFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0, c.Arg1, c.Arg2}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c GitserverClientCommitDateFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1, c.Result2, c.Result3}
}

// GitserverClientCommitsUniqueToBranchFunc describes the behavior when the
// CommitsUniqueToBranch method of the parent MockGitserverClient instance
// is invoked.
type GitserverClientCommitsUniqueToBranchFunc struct {
	defaultHook func(context.Context, int, string, bool, *time.Time) (map[string]time.Time, error)
	hooks       []func(context.Context, int, string, bool, *time.Time) (map[string]time.Time, error)
	history     []GitserverClientCommitsUniqueToBranchFuncCall
	mutex       sync.Mutex
}

// CommitsUniqueToBranch delegates to the next hook function in the queue
// and stores the parameter and result values of this invocation.
func (m *MockGitserverClient) CommitsUniqueToBranch(v0 context.Context, v1 int, v2 string, v3 bool, v4 *time.Time) (map[string]time.Time, error) {
	r0, r1 := m.CommitsUniqueToBranchFunc.nextHook()(v0, v1, v2, v3, v4)
	m.CommitsUniqueToBranchFunc.appendCall(GitserverClientCommitsUniqueToBranchFuncCall{v0, v1, v2, v3, v4, r0, r1})
	return r0, r1
}

// SetDefaultHook sets function that is called when the
// CommitsUniqueToBranch method of the parent MockGitserverClient instance
// is invoked and the hook queue is empty.
func (f *GitserverClientCommitsUniqueToBranchFunc) SetDefaultHook(hook func(context.Context, int, string, bool, *time.Time) (map[string]time.Time, error)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// CommitsUniqueToBranch method of the parent MockGitserverClient instance
// invokes the hook at the front of the queue and discards it. After the
// queue is empty, the default hook function is invoked for any future
// action.
func (f *GitserverClientCommitsUniqueToBranchFunc) PushHook(hook func(context.Context, int, string, bool, *time.Time) (map[string]time.Time, error)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultHook with a function that returns the
// given values.
func (f *GitserverClientCommitsUniqueToBranchFunc) SetDefaultReturn(r0 map[string]time.Time, r1 error) {
	f.SetDefaultHook(func(context.Context, int, string, bool, *time.Time) (map[string]time.Time, error) {
		return r0, r1
	})
}

// PushReturn calls PushHook with a function that returns the given values.
func (f *GitserverClientCommitsUniqueToBranchFunc) PushReturn(r0 map[string]time.Time, r1 error) {
	f.PushHook(func(context.Context, int, string, bool, *time.Time) (map[string]time.Time, error) {
		return r0, r1
	})
}

func (f *GitserverClientCommitsUniqueToBranchFunc) nextHook() func(context.Context, int, string, bool, *time.Time) (map[string]time.Time, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *GitserverClientCommitsUniqueToBranchFunc) appendCall(r0 GitserverClientCommitsUniqueToBranchFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of
// GitserverClientCommitsUniqueToBranchFuncCall objects describing the
// invocations of this function.
func (f *GitserverClientCommitsUniqueToBranchFunc) History() []GitserverClientCommitsUniqueToBranchFuncCall {
	f.mutex.Lock()
	history := make([]GitserverClientCommitsUniqueToBranchFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// GitserverClientCommitsUniqueToBranchFuncCall is an object that describes
// an invocation of method CommitsUniqueToBranch on an instance of
// MockGitserverClient.
type GitserverClientCommitsUniqueToBranchFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 context.Context
	// Arg1 is the value of the 2nd argument passed to this method
	// invocation.
	Arg1 int
	// Arg2 is the value of the 3rd argument passed to this method
	// invocation.
	Arg2 string
	// Arg3 is the value of the 4th argument passed to this method
	// invocation.
	Arg3 bool
	// Arg4 is the value of the 5th argument passed to this method
	// invocation.
	Arg4 *time.Time
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 map[string]time.Time
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 error
}

// Args returns an interface slice containing the arguments of this
// invocation.
func (c GitserverClientCommitsUniqueToBranchFuncCall) Args() []interface{} {
	return []interface{}{c.Arg0, c.Arg1, c.Arg2, c.Arg3, c.Arg4}
}

// Results returns an interface slice containing the results of this
// invocation.
func (c GitserverClientCommitsUniqueToBranchFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1}
}

// GitserverClientRefDescriptionsFunc describes the behavior when the
// RefDescriptions method of the parent MockGitserverClient instance is
// invoked.
type GitserverClientRefDescriptionsFunc struct {
	defaultHook func(context.Context, int, ...string) (map[string][]gitdomain.RefDescription, error)
	hooks       []func(context.Context, int, ...string) (map[string][]gitdomain.RefDescription, error)
	history     []GitserverClientRefDescriptionsFuncCall
	mutex       sync.Mutex
}

// RefDescriptions delegates to the next hook function in the queue and
// stores the parameter and result values of this invocation.
func (m *MockGitserverClient) RefDescriptions(v0 context.Context, v1 int, v2 ...string) (map[string][]gitdomain.RefDescription, error) {
	r0, r1 := m.RefDescriptionsFunc.nextHook()(v0, v1, v2...)
	m.RefDescriptionsFunc.appendCall(GitserverClientRefDescriptionsFuncCall{v0, v1, v2, r0, r1})
	return r0, r1
}

// SetDefaultHook sets function that is called when the RefDescriptions
// method of the parent MockGitserverClient instance is invoked and the hook
// queue is empty.
func (f *GitserverClientRefDescriptionsFunc) SetDefaultHook(hook func(context.Context, int, ...string) (map[string][]gitdomain.RefDescription, error)) {
	f.defaultHook = hook
}

// PushHook adds a function to the end of hook queue. Each invocation of the
// RefDescriptions method of the parent MockGitserverClient instance invokes
// the hook at the front of the queue and discards it. After the queue is
// empty, the default hook function is invoked for any future action.
func (f *GitserverClientRefDescriptionsFunc) PushHook(hook func(context.Context, int, ...string) (map[string][]gitdomain.RefDescription, error)) {
	f.mutex.Lock()
	f.hooks = append(f.hooks, hook)
	f.mutex.Unlock()
}

// SetDefaultReturn calls SetDefaultHook with a function that returns the
// given values.
func (f *GitserverClientRefDescriptionsFunc) SetDefaultReturn(r0 map[string][]gitdomain.RefDescription, r1 error) {
	f.SetDefaultHook(func(context.Context, int, ...string) (map[string][]gitdomain.RefDescription, error) {
		return r0, r1
	})
}

// PushReturn calls PushHook with a function that returns the given values.
func (f *GitserverClientRefDescriptionsFunc) PushReturn(r0 map[string][]gitdomain.RefDescription, r1 error) {
	f.PushHook(func(context.Context, int, ...string) (map[string][]gitdomain.RefDescription, error) {
		return r0, r1
	})
}

func (f *GitserverClientRefDescriptionsFunc) nextHook() func(context.Context, int, ...string) (map[string][]gitdomain.RefDescription, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if len(f.hooks) == 0 {
		return f.defaultHook
	}

	hook := f.hooks[0]
	f.hooks = f.hooks[1:]
	return hook
}

func (f *GitserverClientRefDescriptionsFunc) appendCall(r0 GitserverClientRefDescriptionsFuncCall) {
	f.mutex.Lock()
	f.history = append(f.history, r0)
	f.mutex.Unlock()
}

// History returns a sequence of GitserverClientRefDescriptionsFuncCall
// objects describing the invocations of this function.
func (f *GitserverClientRefDescriptionsFunc) History() []GitserverClientRefDescriptionsFuncCall {
	f.mutex.Lock()
	history := make([]GitserverClientRefDescriptionsFuncCall, len(f.history))
	copy(history, f.history)
	f.mutex.Unlock()

	return history
}

// GitserverClientRefDescriptionsFuncCall is an object that describes an
// invocation of method RefDescriptions on an instance of
// MockGitserverClient.
type GitserverClientRefDescriptionsFuncCall struct {
	// Arg0 is the value of the 1st argument passed to this method
	// invocation.
	Arg0 context.Context
	// Arg1 is the value of the 2nd argument passed to this method
	// invocation.
	Arg1 int
	// Arg2 is a slice containing the values of the variadic arguments
	// passed to this method invocation.
	Arg2 []string
	// Result0 is the value of the 1st result returned from this method
	// invocation.
	Result0 map[string][]gitdomain.RefDescription
	// Result1 is the value of the 2nd result returned from this method
	// invocation.
	Result1 error
}

// Args returns an interface slice containing the arguments of this
// invocation. The variadic slice argument is flattened in this array such
// that one positional argument and three variadic arguments would result in
// a slice of four, not two.
func (c GitserverClientRefDescriptionsFuncCall) Args() []interface{} {
	trailing := []interface{}{}
	for _, val := range c.Arg2 {
		trailing = append(trailing, val)
	}

	return append([]interface{}{c.Arg0, c.Arg1}, trailing...)
}

// Results returns an interface slice containing the results of this
// invocation.
func (c GitserverClientRefDescriptionsFuncCall) Results() []interface{} {
	return []interface{}{c.Result0, c.Result1}
}
