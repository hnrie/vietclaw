package memory

import "database/sql"

func scanRecords(rows *sql.Rows) ([]Record, error) {
	records := []Record{}
	for rows.Next() {
		var rec Record
		var kind string
		var confidence float64
		var embBytes []byte
		if err := rows.Scan(&rec.ID, &rec.Scope, &kind, &rec.Content, &confidence, &rec.CreatedAt, &rec.UpdatedAt, &embBytes); err != nil {
			// Fallback if schema doesn't have embedding column queried yet in standard query helper
			if errScan := rows.Scan(&rec.ID, &rec.Scope, &kind, &rec.Content, &confidence, &rec.CreatedAt, &rec.UpdatedAt); errScan == nil {
				rec.Kind = Kind(kind)
				rec.Confidence = confidenceLabel(confidence)
				records = append(records, rec)
				continue
			}
			return nil, err
		}
		rec.Kind = Kind(kind)
		rec.Confidence = confidenceLabel(confidence)
		if len(embBytes) > 0 {
			rec.Embedding = BytesToFloat32Slice(embBytes)
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}
