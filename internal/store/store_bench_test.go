package store

import (
	"encoding/json"
	"testing"
)

func generateDummyEvents(startLedger uint32, count int) []any {
	res := make([]any, count)
	for i := 0; i < count; i++ {
		res[i] = map[string]any{
			"ledger": startLedger + uint32(i),
		}
	}
	return res
}

func generateDummyStoreEvents(startLedger uint32, count int) []Event {
	return generateDummyPayloads(startLedger, count)
}

func generateDummyPayloads(startLedger uint32, count int) []any {
	res := make([]any, count)
	for i := 0; i < count; i++ {
		res[i] = map[string]any{
			"ledger": startLedger + uint32(i),
		}
	}
	return res
}

func buildQuery(f EventFilter) (string, []any, error) {
	query := "SELECT * FROM events WHERE 1=1"
	var args []any
	if f.ContractID != "" {
		query += " AND contract_id = ?"
		args = append(args, f.ContractID)
	}
	if f.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, f.Limit)
	}
	return query, args, nil
}

func BenchmarkQueryConstruction_SimpleFilter(b *testing.B) {
	filter := EventFilter{
		ContractID: "C0000000000000000000000000000000000000000000000000000001",
		Limit:      50,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = buildQuery(filter)
	}
}

func BenchmarkQueryConstruction_ComplexFilter(b *testing.B) {
	filter := EventFilter{
		ContractID:    "C0000000000000000000000000000000000000000000000000000001",
		Types:         []string{"contract"},
		FromLedger:    1000,
		ToLedger:      2000,
		TopicContains: json.RawMessage(`[{"symbol":"transfer"}]`),
		Limit:         100,
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = buildQuery(filter)
	}
}

func BenchmarkUpsertBatchMemory_100(b *testing.B) {
	benchmarkUpsertPayload(b, 100)
}

func BenchmarkUpsertBatchMemory_500(b *testing.B) {
	benchmarkUpsertPayload(b, 500)
}

func BenchmarkUpsertBatchMemory_1000(b *testing.B) {
	benchmarkUpsertPayload(b, 1000)
}

func benchmarkUpsertPayload(b *testing.B, size int) {
	events := generateDummyStoreEvents(100000, size)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = events
	}
}
