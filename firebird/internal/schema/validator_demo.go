package schema

// This file demonstrates the proposed validation pipeline API design for Phase 1A.
// It shows the key interfaces and types without full implementation.

/*

PROPOSED API - Validation Pipeline with Foreign Key Detection

## Core Types:

type ValidatorResult struct {
	Errors   []ValidationError  // Blocking errors
	Warnings []ValidationError  // Non-blocking warnings
	Infos    []ValidationError  // Informational messages
}

type Validator interface {
	Name() string
	Validate(def *Definition, lineMap map[string]int) (ValidatorResult, error)
}

type ValidationPipeline struct {
	validators  []Validator
	interactive bool
}

## Usage Example:

```go
// Create validation pipeline (interactive mode prompts for FK constraints)
pipeline := schema.NewValidationPipeline(true)

// Run validation
result, err := pipeline.Validate(def, lineMap)
if err != nil {
	return err
}

// Check for errors
if result.HasErrors() {
	fmt.Println(result.Error())  // Pretty-printed with colors and suggestions
	return errors.New("validation failed")
}

// Warnings and infos are printed but don't block generation
```

## Built-in Validators:

1. **FieldNameValidator** - Detects:
   - Reserved words (id, created_at, updated_at)
   - Go keywords (select, type, func, etc.)
   - SQL keywords (select, insert, where, etc.)

2. **TypeValidator** - Validates:
   - Go type / db_type compatibility
   - string → TEXT/VARCHAR
   - int64 → INTEGER/BIGINT
   - Suggests correct types when mismatch detected

3. **ForeignKeyDetector** - Detects FK patterns:
   - Fields ending in _id with int64 type
   - Interactive: Prompts "Add FK constraint to posts.id? [Y/n]"
   - Prompts for ON DELETE behavior (CASCADE/RESTRICT/SET NULL/NO ACTION)
   - Stores FK metadata in field.Tags["fk"], field.Tags["fk_on_delete"]
   - Non-interactive: Warns about potential FKs

4. **RelationshipValidator** - Validates:
   - FK fields have constraints configured
   - FK naming follows conventions
   - Future: Cross-schema validation (referenced tables exist)

## Foreign Key Metadata Storage:

FK metadata is stored in field tags:

```yaml
- name: post_id
  type: int64
  db_type: INTEGER
  tags:
    fk: "posts.id"              # Target table.column
    fk_on_delete: "CASCADE"     # ON DELETE behavior
    fk_on_update: "CASCADE"     # ON UPDATE behavior
```

## Sample Output:

```
Validating schema...

🔍 Detected potential foreign key: comments.post_id → posts.id
   Add FK constraint? [Y/n]: y
   Select ON DELETE behavior:
   1. CASCADE (delete dependent records)
   2. RESTRICT (prevent deletion if references exist)
   3. SET NULL (set FK to NULL)
   4. NO ACTION (same as RESTRICT)
   Enter choice [1-4]: 1

✓ Field names valid
✓ Types compatible
ℹ Added FK constraint: post_id → posts.id (ON DELETE CASCADE)

✨ Validation passed!
```

## Integration Points:

1. **cmd/generate/generate.go** (line ~280):
   Before generating any code, run validation:

   ```go
   // Validate schema before generation
   pipeline := schema.NewValidationPipeline(true) // interactive mode
   result, err := pipeline.Validate(def, lineMap)
   if err != nil {
       return err
   }

   if result.HasErrors() {
       fmt.Println(result.Error())
       os.Exit(1)
   }
   ```

2. **internal/generators/migration/migration.go** (line ~115):
   Generate FK constraints in CREATE TABLE:

   ```go
   // Extract FK metadata from field tags
   for _, field := range def.Spec.Fields {
       if fk := field.Tags["fk"]; fk != "" {
           onDelete := field.Tags["fk_on_delete"]
           // Add to migration: FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE
       }
   }
   ```

3. **Migration Ordering** (new function):
   Detect dependencies and order migrations:

   ```go
   func OrderMigrations(defs []*Definition) ([]*Definition, error) {
       // Build dependency graph from FK constraints
       // Topological sort
       // Return ordered list (dependencies first)
   }
   ```

## Next Steps (Phase 1B):

1. Implement FK constraint generation in migration templates
2. Add migration ordering logic for multi-resource generation
3. Write unit tests for each validator
4. Add integration test: multi-resource schema with FKs generates correctly

*/
