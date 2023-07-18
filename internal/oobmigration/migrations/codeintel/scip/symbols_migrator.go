package scip

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/keegancsmith/sqlf"
	"github.com/lib/pq"

	"github.com/sourcegraph/sourcegraph/internal/codeintel/shared/symbols"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/batch"
	"github.com/sourcegraph/sourcegraph/internal/database/dbutil"
)

func getEnv(name string, defaultValue int) int {
	if value, _ := strconv.Atoi(os.Getenv(name)); value != 0 {
		return value
	}

	return defaultValue
}

var (
	// NOTE: modified in tests
	symbolsMigratorConcurrencyLevel      = getEnv("SYMBOLS_MIGRATOR_CONCURRENCY_LEVEL", 1)
	symbolsMigratorSymbolRecordBatchSize = getEnv("SYMBOLS_MIGRATOR_UPLOAD_BATCH_SIZE", 25000)
)

type scipSymbolsMigrator struct {
	codeintelStore *basestore.Store
}

func NewSCIPSymbolsMigrator(codeintelStore *basestore.Store) *migrator {
	driver := &scipSymbolsMigrator{
		codeintelStore: codeintelStore,
	}

	return newMigrator(codeintelStore, driver, migratorOptions{
		tableName:     "codeintel_scip_symbols",
		targetVersion: 2,
		batchSize:     symbolsMigratorSymbolRecordBatchSize,
		numRoutines:   symbolsMigratorConcurrencyLevel,
		fields: []fieldSpec{
			{name: "symbol_id", postgresType: "integer not null", primaryKey: true},
			{name: "document_lookup_id", postgresType: "integer not null", primaryKey: true},
		},
	})
}

func (m *scipSymbolsMigrator) ID() int                 { return 24 }
func (m *scipSymbolsMigrator) Interval() time.Duration { return time.Second }

func (m *scipSymbolsMigrator) MigrateUp(ctx context.Context, uploadID int, tx *basestore.Store, rows *sql.Rows) (_ [][]any, err error) {
	fmt.Printf(">\n")
	start := time.Now()
	lapTimer := start
	lapTimes := map[string]time.Duration{}
	lap := func(format string, args ...any) {
		now := time.Now()
		lapTimes[fmt.Sprintf(format, args...)] = now.Sub(lapTimer)
		lapTimer = now
	}
	done := func(format string, args ...any) {
		lapNames := make([]string, 0, len(lapTimes))
		for lapName := range lapTimes {
			lapNames = append(lapNames, lapName)
		}
		sort.Slice(lapNames, func(i, j int) bool { return lapTimes[lapNames[i]] > lapTimes[lapNames[j]] })

		for _, name := range lapNames {
			fmt.Printf("\t%s:\t%s\n", lapTimes[name], name)
		}
		fmt.Printf("> %s:\t%s\n\n", time.Since(start), fmt.Sprintf(format, args...))
	}

	// Consume symbol_id/document_id pairs from the incoming rows
	symbolInDocuments, err := readSymbolInDocuments(rows)
	if err != nil {
		return nil, err
	}
	symbolIDMap := make(map[int]struct{}, len(symbolInDocuments))
	for _, pair := range symbolInDocuments {
		symbolIDMap[pair.symbolID] = struct{}{}
	}
	symbolIDs := flattenKeys(symbolIDMap)

	lap("read %d rows", len(symbolInDocuments))

	// Reconstruct the full symbol names for each of the symbol IDs in this batch
	symbolNamesByID, err := readSymbolNamesBySymbolIDs(ctx, tx, uploadID, symbolIDs)
	if err != nil {
		return nil, err
	}
	lap("read %d symbol names", len(symbolNamesByID))
	symbolNames := flattenValues(symbolNamesByID)
	lap("flatten %d symbol names", len(symbolNames))

	// An upload's symbols may be processed over several batches, and each symbol
	// identifier needs to be unique per upload, so we track the highest symbol
	// identifier written by the migration per upload. We read the last written
	// value here (or default zero), and write our next highest identifier upon
	// (successful) exit of this method.
	nextSymbolLookupID, err := getNextSymbolID(ctx, tx, uploadID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err == nil {
			err = setNextSymbolID(ctx, tx, uploadID, nextSymbolLookupID)
		}
	}()

	// Convert symbol names into a tree structure we'll insert into the database
	// All identifiers here are created ahead of the insertion so we do not need
	// to do multiple round-trips to get new insertion identifiers for pending
	// data - everything is known up-front.

	id := func() int { id := nextSymbolLookupID; nextSymbolLookupID++; return id }
	cache, traverser, err := constructSymbolLookupTable(symbolNames, id)
	if err != nil {
		return nil, err
	}

	lap("constructed cache of %d items", len(cache))
	buffer := make([][]any, 0, len(symbolNames)*2)

	// EXTRACTED FOR DEBUGGING
	visit := func(segmentType string, segmentQuality *string, name string, id int, parentID *int) error {
		buffer = append(buffer, []any{segmentType, segmentQuality, name, id, parentID})
		return nil
	}
	if err := traverser(visit); err != nil {
		return nil, err
	}

	lap("prepared %d lookups", len(buffer))
	rowsX := 0

	// Bulk insert the content of the tree / descriptor-no-suffix map
	symbolNamePartInserter := func(ctx context.Context, symbolLookupInserter *batch.Inserter) error {
		for _, values := range buffer {
			rowsX++
			if err := symbolLookupInserter.Insert(ctx, values...); err != nil {
				return err
			}
		}

		return nil
	}

	// In the same transaction but ouf-of-band from the row updates, batch-insert new
	// symbol-lookup rows. These identifiers need to exist before the batch is complete.
	if err := withSymbolLookupInserter(ctx, tx, uploadID, symbolNamePartInserter, func(format string, args ...any) { lap("lookups: "+format, args...) }); err != nil {
		return nil, err
	}

	lap("inserted %d lookups", rowsX)
	rowsY := 0

	// Bulk insert descriptor/descriptor-no-suffix pairs with relations to their symbol
	symbolRelationshipInserter := func(ctx context.Context, symbolLookupLeavesInserter *batch.Inserter) error {
		for _, symbolInDocument := range symbolInDocuments {
			symbolID := symbolInDocument.symbolID
			ids := cache[symbolNamesByID[symbolID]]

			rowsY++
			if err := symbolLookupLeavesInserter.Insert(ctx, symbolID, ids.descriptorSuffixID, ids.fuzzyDescriptorSuffixID); err != nil {
				return err
			}
		}

		return nil
	}

	// In the same transaction but ouf-of-band from the row updates, batch-insert new
	// symbol-lookup-leaves rows. These identifiers need to exist before the batch is complete.
	if err := withSymbolLookupLeavesInserter(ctx, tx, uploadID, symbolRelationshipInserter, func(format string, args ...any) { lap("leaves: "+format, args...) }); err != nil {
		return nil, err
	}

	lap("inserted %d leaves", rowsY)

	// Construct the updated tuples for the symbols rows we have locked in this transaction.
	// Each (original) symbol identifier is translated into the new descriptor identifier
	// pairs batch inserted into the symbols lookup table (above). We're not supplying any
	// additional information here, so we'll end up just writing foreign keys _to_ these rows
	// and bumping the schema version.
	values := make([][]any, 0, len(symbolInDocuments))
	for _, symbolInDocument := range symbolInDocuments {
		values = append(values, []any{
			symbolInDocument.symbolID,
			symbolInDocument.documentLookupID,
		})
	}

	done("updated %d rows and inserted %d symbols", len(values), len(symbolNames))
	return values, nil
}

//
//
//

func (m *scipSymbolsMigrator) MigrateDown(ctx context.Context, uploadID int, tx *basestore.Store, rows *sql.Rows) (_ [][]any, err error) {
	// Consume symbol_id/document_id pairs from the incoming rows
	symbolInDocuments, err := readSymbolInDocuments(rows)
	if err != nil {
		return nil, err
	}

	// Remove the keys we added in the up direction
	values := make([][]any, 0, len(symbolInDocuments))
	for _, symbolInDocument := range symbolInDocuments {
		values = append(values, []any{
			symbolInDocument.symbolID,
			symbolInDocument.documentLookupID,
			nil,
			nil,
		})
	}

	return values, nil
}

//
//
//

type symbolInDocument struct {
	symbolID         int
	documentLookupID int
}

func readSymbolInDocuments(rows *sql.Rows) ([]symbolInDocument, error) {
	return scanSymbolInDocuments(rows, nil)
}

var scanSymbolInDocuments = basestore.NewSliceScanner(func(s dbutil.Scanner) (sd symbolInDocument, _ error) {
	err := s.Scan(&sd.symbolID, &sd.documentLookupID)
	return sd, err
})

//
//

func readSymbolNamesBySymbolIDs(ctx context.Context, tx *basestore.Store, uploadID int, symbolIDs []int) (map[int]string, error) {
	return scanSymbolNamesByID(tx.Query(ctx, sqlf.Sprintf(`
		WITH RECURSIVE
		symbols(id, upload_id, suffix, prefix_id) AS (
			(
				SELECT
					ssn.id,
					ssn.upload_id,
					ssn.name_segment AS suffix,
					ssn.prefix_id
				FROM codeintel_scip_symbol_names ssn
				WHERE
					ssn.id = ANY(%s) AND
					ssn.upload_id = %s
			) UNION (
				SELECT
					s.id,
					s.upload_id,
					ssn.name_segment || s.suffix AS suffix,
					ssn.prefix_id
				FROM symbols s
				JOIN codeintel_scip_symbol_names ssn ON
					ssn.upload_id = s.upload_id AND
					ssn.id = s.prefix_id
			)
		)
		SELECT s.id, s.suffix AS symbol_name
		FROM symbols s
		WHERE s.prefix_id IS NULL
	`,
		pq.Array(symbolIDs),
		uploadID,
	)))
}

var scanSymbolNamesByID = basestore.NewMapScanner(func(s dbutil.Scanner) (symbolID int, symbolName string, _ error) {
	err := s.Scan(&symbolID, &symbolName)
	return symbolID, symbolName, err
})

//
//

func getNextSymbolID(ctx context.Context, tx *basestore.Store, uploadID int) (int, error) {
	nextSymbolLookupID, _, err := basestore.ScanFirstInt(tx.Query(ctx, sqlf.Sprintf(`
		SELECT symbol_id
		FROM codeintel_scip_symbols_migration_progress
		WHERE upload_id = %s
	`,
		uploadID,
	)))
	return nextSymbolLookupID, err
}

func setNextSymbolID(ctx context.Context, tx *basestore.Store, uploadID, id int) error {
	return tx.Exec(ctx, sqlf.Sprintf(`
		INSERT INTO codeintel_scip_symbols_migration_progress (upload_id, symbol_id)
		VALUES (%s, %s)
		ON CONFLICT (upload_id) DO UPDATE SET symbol_id = EXCLUDED.symbol_id
	`,
		uploadID,
		id,
	))
}

//
//

type inserterFunc func(ctx context.Context, symbolLookupInserter *batch.Inserter) error

func withSymbolLookupInserter(ctx context.Context, tx *basestore.Store, uploadID int, f inserterFunc, lap func(format string, args ...any)) error {
	if err := tx.Exec(ctx, sqlf.Sprintf(`
		CREATE TEMPORARY TABLE t_codeintel_scip_symbols_lookup(
			name text NOT NULL,
			id integer NOT NULL,
			segment_type SymbolNameSegmentType NOT NULL,
			segment_quality SymbolNameSegmentQuality,
			parent_id integer
		) ON COMMIT DROP
	`)); err != nil {
		return err
	}

	symbolLookupInserter := batch.NewInserter(
		ctx,
		tx.Handle(),
		"t_codeintel_scip_symbols_lookup",
		batch.MaxNumPostgresParameters,
		"segment_type",
		"segment_quality",
		"name",
		"id",
		"parent_id",
	)

	if err := f(ctx, symbolLookupInserter); err != nil {
		return err
	}
	if err := symbolLookupInserter.Flush(ctx); err != nil {
		return err
	}

	lap("flushed %d times", symbolLookupInserter.NumFlushes)

	if err := tx.Exec(ctx, sqlf.Sprintf(`
		INSERT INTO codeintel_scip_symbols_lookup (id, upload_id, name, segment_type, segment_quality, parent_id)
		SELECT id, %s, name, segment_type, segment_quality, parent_id
		FROM t_codeintel_scip_symbols_lookup
	`,
		uploadID,
	)); err != nil {
		return err
	}

	lap("move temp")
	return nil
}

func withSymbolLookupLeavesInserter(ctx context.Context, tx *basestore.Store, uploadID int, f inserterFunc, lap func(format string, args ...any)) error {
	if err := tx.Exec(ctx, sqlf.Sprintf(`
		CREATE TEMPORARY TABLE t_codeintel_scip_symbols_lookup_leaves(
			symbol_id integer NOT NULL,
			descriptor_suffix_id integer NOT NULL,
			fuzzy_descriptor_suffix_id integer NOT NULL
		) ON COMMIT DROP
	`)); err != nil {
		return err
	}

	symbolLookupLeavesInserter := batch.NewInserter(
		ctx,
		tx.Handle(),
		"t_codeintel_scip_symbols_lookup_leaves",
		batch.MaxNumPostgresParameters,
		"symbol_id",
		"descriptor_suffix_id",
		"fuzzy_descriptor_suffix_id",
	)

	if err := f(ctx, symbolLookupLeavesInserter); err != nil {
		return err
	}
	if err := symbolLookupLeavesInserter.Flush(ctx); err != nil {
		return err
	}

	lap("flushed %d times", symbolLookupLeavesInserter.NumFlushes)

	if err := tx.Exec(ctx, sqlf.Sprintf(`
		INSERT INTO codeintel_scip_symbols_lookup_leaves (upload_id, symbol_id, descriptor_suffix_id, fuzzy_descriptor_suffix_id)
		SELECT %s, symbol_id, descriptor_suffix_id, fuzzy_descriptor_suffix_id
		FROM t_codeintel_scip_symbols_lookup_leaves
	`,
		uploadID,
	)); err != nil {
		return err
	}

	lap("move temp")
	return nil
}

//
//

func flattenKeys[K int | string, V any](m map[K]V) []K {
	ss := make([]K, 0, len(m))
	for v := range m {
		ss = append(ss, v)
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i] < ss[j] })

	return ss
}

func flattenValues[K comparable, V int | string](m map[K]V) []V {
	ss := make([]V, 0, len(m))
	for _, v := range m {
		ss = append(ss, v)
	}
	sort.Slice(ss, func(i, j int) bool { return ss[i] < ss[j] })

	return ss
}

// NOTE(scip-migration): This behavior in the following function has been copied from the upload
// processing procedure to match the logic for newly uploaded index files. See the original code
// in the uploads service for more detail (.../lsifstore/insert.go).

type explodedIDs struct {
	descriptorSuffixID      int
	fuzzyDescriptorSuffixID int
}

type visitFunc func(segmentType string, segmentQuality *string, name string, id int, parentID *int) error

func constructSymbolLookupTable(symbolNames []string, id func() int) (map[string]explodedIDs, func(visit visitFunc) error, error) {
	cache := map[string]explodedIDs{}     // Tracks symbol name -> identifiers in the scheme tree
	schemeTree := map[string]SchemeNode{} // Tracks scheme -> manager -> name -> version -> descriptor namespace -> descriptor suffix
	qualityMap := map[int]string{}        // Tracks descriptor node ids -> quality (PRECISE, FUZZY, BOTH)

	// Create helpers to create new tree nodes with (upload-)unique identifiers
	createSchemeNode := func() SchemeNode { return SchemeNode(newNodeWithID[PackageManagerNode](id())) }
	createPackageManagerNode := func() PackageManagerNode { return PackageManagerNode(newNodeWithID[PackageNameNode](id())) }
	createPackageNameNode := func() PackageNameNode { return PackageNameNode(newNodeWithID[PackageVersionNode](id())) }
	createPackageVersionNode := func() PackageVersionNode { return PackageVersionNode(newNodeWithID[NamespaceNode](id())) }
	createNamespaceNode := func() NamespaceNode { return NamespaceNode(newNodeWithID[DescriptorNode](id())) }
	createDescriptorWithQuality := func(quality string) func() DescriptorNode {
		return func() DescriptorNode {
			id := id()
			qualityMap[id] = quality
			return DescriptorNode(newNodeWithID[descriptor](id))
		}
	}
	createPreciseDescriptor := createDescriptorWithQuality("PRECISE")
	createFuzzyDescriptor := createDescriptorWithQuality("FUZZY")
	createUnionDescriptor := createDescriptorWithQuality("BOTH")

	for _, symbolName := range symbolNames {
		symbol, err := symbols.NewExplodedSymbol(symbolName)
		if err != nil {
			return nil, nil, err
		}

		// Assign the parts of the exploded symbol into the scheme tree. If a prefix of the exploded symbol
		// is already in the tree then existing nodes will be re-used. Laying out the exploded in a tree
		// structure will allow us to trace parentage (required for fast lookups) when we insert these into
		// the database.

		schemeNode := getOrCreate(schemeTree, symbol.Scheme, createSchemeNode)                                       // depth 0
		packageManagerNode := getOrCreate(schemeNode.children, symbol.PackageManager, createPackageManagerNode)      // depth 1
		packageNameNode := getOrCreate(packageManagerNode.children, symbol.PackageName, createPackageNameNode)       // depth 2
		packageVersionNode := getOrCreate(packageNameNode.children, symbol.PackageVersion, createPackageVersionNode) // depth 3
		namespace := getOrCreate(packageVersionNode.children, symbol.DescriptorNamespace, createNamespaceNode)       // depth 4
		descriptorSuffixID, fuzzyDescriptorSuffixID := getOrCreateLeafIDs(                                           // depth 5
			namespace,
			symbol.DescriptorSuffix, symbol.FuzzyDescriptorSuffix,
			createPreciseDescriptor, createFuzzyDescriptor, createUnionDescriptor,
		)

		cache[symbolName] = explodedIDs{
			descriptorSuffixID:      descriptorSuffixID,
			fuzzyDescriptorSuffixID: fuzzyDescriptorSuffixID,
		}
	}

	segmentTypeByDepth := []string{
		"SCHEME",               // depth 0
		"PACKAGE_MANAGER",      // depth 1
		"PACKAGE_NAME",         // depth 2
		"PACKAGE_VERSION",      // depth 3
		"DESCRIPTOR_NAMESPACE", // depth 4
		"DESCRIPTOR_SUFFIX",    // depth 5
		/*                   */ // depth PANIC
	}
	segmentQualityForID := func(id int) *string {
		if quality, ok := qualityMap[id]; ok {
			return &quality
		}

		return nil
	}

	traverser := func(visit visitFunc) error {
		if err := traverse(schemeTree, func(name string, id, depth int, parentID *int) error {
			return visit(segmentTypeByDepth[depth], segmentQualityForID(id), name, id, parentID)
		}); err != nil {
			return err
		}

		return nil
	}

	return cache, traverser, nil
}

func getOrCreateLeafIDs(
	namespace treeNode[treeNode[descriptor]],
	descriptorSuffix, fuzzyDescriptorSuffix string,
	createPreciseDescriptor, createFuzzyDescriptor, createUnionDescriptor func() treeNode[descriptor],
) (int, int) {
	if descriptorSuffix == fuzzyDescriptorSuffix {
		// Common case: no difference - create a single leaf node
		descriptor := getOrCreate(namespace.children, descriptorSuffix, createUnionDescriptor)
		return descriptor.id, descriptor.id
	}

	// General case: unique fuzzy descriptor - create two leaf nodes
	descriptor := getOrCreate(namespace.children, descriptorSuffix, createPreciseDescriptor)
	fuzzyDescriptor := getOrCreate(namespace.children, fuzzyDescriptorSuffix, createFuzzyDescriptor)
	return descriptor.id, fuzzyDescriptor.id
}
