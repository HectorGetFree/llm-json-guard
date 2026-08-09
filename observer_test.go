package llmjsonguard

import (
	"context"
	"testing"
)

func TestParseObserverRecordsDirectSuccess(t *testing.T) {
	raw := `{"name":"Alice","age":18,"tags":[],"active":true}`
	var observations []ParseObservation
	result, err := Parse[Person](context.Background(), raw, ParseOptions[Person]{
		Schema: personSchema,
		Observer: func(_ context.Context, observation ParseObservation) {
			observations = append(observations, observation)
		},
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result.Path != ParsePathDirect || len(observations) != 1 {
		t.Fatalf("result = %#v, observations = %#v", result, observations)
	}
	observation := observations[0]
	if !observation.Success || observation.Path != ParsePathDirect {
		t.Fatalf("unexpected observation: %#v", observation)
	}
	if observation.InputBytes != len(raw) || observation.SchemaBytes != len(personSchema) {
		t.Fatalf("unexpected byte metrics: %#v", observation)
	}
	if observation.CandidateCount != 0 || observation.LocalRepairAttempts != 0 || observation.LLMRepairCalls != 0 {
		t.Fatalf("direct path must not report recovery work: %#v", observation)
	}
}

func TestParseObserverRecordsLocalRepair(t *testing.T) {
	var observation ParseObservation
	result, err := Parse[Person](context.Background(), `{name:'Alice',age:18,tags:[],active:true,}`, ParseOptions[Person]{
		Schema: personSchema,
		Observer: func(_ context.Context, value ParseObservation) {
			observation = value
		},
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result.Path != ParsePathLocalFix || observation.Path != ParsePathLocalFix {
		t.Fatalf("result = %#v, observation = %#v", result, observation)
	}
	if observation.CandidateCount == 0 || observation.LocalRepairAttempts != 1 || observation.LLMRepairCalls != 0 {
		t.Fatalf("unexpected recovery metrics: %#v", observation)
	}
}

func TestParseObserverRecordsLLMRepair(t *testing.T) {
	var observation ParseObservation
	result, err := Parse[Person](context.Background(), "not json", ParseOptions[Person]{
		Schema: personSchema,
		LLMRepair: func(context.Context, LLMRepairRequest) (string, error) {
			return `{"name":"Alice","age":18,"tags":[],"active":true}`, nil
		},
		Observer: func(_ context.Context, value ParseObservation) {
			observation = value
		},
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result.Path != ParsePathLLMFix || observation.LLMRepairCalls != 1 || observation.LocalRepairAttempts != 0 {
		t.Fatalf("result = %#v, observation = %#v", result, observation)
	}
}

func TestParseObserverRecordsFailureOnce(t *testing.T) {
	calls := 0
	var observation ParseObservation
	_, err := Parse[Person](context.Background(), "no structured output", ParseOptions[Person]{
		Schema: personSchema,
		Observer: func(_ context.Context, value ParseObservation) {
			calls++
			observation = value
		},
	})
	if err == nil {
		t.Fatal("expected Parse to fail")
	}
	if calls != 1 || observation.Success {
		t.Fatalf("calls = %d, observation = %#v", calls, observation)
	}
	if observation.ErrorCode != ErrorCodeNoCandidate || observation.Stage != ParseStageExtraction {
		t.Fatalf("unexpected failure metrics: %#v", observation)
	}
}

func TestParseObserverPanicDoesNotChangeResult(t *testing.T) {
	result, err := Parse[int](context.Background(), "1", ParseOptions[int]{
		Schema: `{"type":"integer"}`,
		Observer: func(context.Context, ParseObservation) {
			panic("observer failed")
		},
	})
	if err != nil || result.Value != 1 {
		t.Fatalf("observer panic changed Parse result: result=%#v err=%v", result, err)
	}
}
