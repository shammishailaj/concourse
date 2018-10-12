package voyager_test

import (
	"github.com/concourse/concourse/atc/db/migration/voyager"
	"github.com/concourse/concourse/atc/db/migration/voyager/voyagerfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var basicSQLMigration = []byte(`
		BEGIN;
		CREATE TABLE some_table;
		COMMIT;`)

var noTransactionMigration = []byte(`
		-- NO_TRANSACTION
		CREATE TYPE enum_type AS ENUM ('blue_type', 'green_type');
		ALTER TYPE enum_type ADD VALUE 'some_type'; `)

var multipleStatementMigration = []byte(`
		BEGIN;
		CREATE TABLE some_table (ID integer, something varchar);
		ALTER TABLE some_table ADD COLUMN notes varchar;
		COMMIT;`)

var sqlFunctionMigration = []byte(`
BEGIN;
  CREATE OR REPLACE FUNCTION on_item_delete() RETURNS TRIGGER AS $$
  BEGIN
          EXECUTE format('DROP TABLE IF EXISTS item%s', OLD.id);
          RETURN NULL;
  END;
  $$ LANGUAGE plpgsql;`)

var _ = Describe("Parser", func() {
	var (
		parser  *voyager.Parser
		bindata *voyagerfakes.FakeSource
	)

	BeforeEach(func() {
		bindata = new(voyagerfakes.FakeSource)
		bindata.AssetReturns([]byte{}, nil)

		parser = voyager.NewParser(bindata)
	})
	It("parses the direction of the migration from the file name", func() {
		downMigration, err := parser.ParseFileToMigration("2000_some_migration.down.go")
		Expect(err).ToNot(HaveOccurred())
		Expect(downMigration.Direction).To(Equal("down"))

		upMigration, err := parser.ParseFileToMigration("1000_some_migration.up.sql")
		Expect(err).ToNot(HaveOccurred())
		Expect(upMigration.Direction).To(Equal("up"))
	})

	It("parses the strategy of the migration from the file", func() {
		downMigration, err := parser.ParseFileToMigration("2000_some_migration.down.go")
		Expect(err).ToNot(HaveOccurred())
		Expect(downMigration.Strategy).To(Equal(voyager.GoMigration))

		bindata.AssetReturns(basicSQLMigration, nil)
		upMigration, err := parser.ParseFileToMigration("1000_some_migration.up.sql")
		Expect(err).ToNot(HaveOccurred())
		Expect(upMigration.Strategy).To(Equal(voyager.SQLTransaction))

		bindata.AssetReturns(noTransactionMigration, nil)
		upNoTxMigration, err := parser.ParseFileToMigration("3000_some_no_transaction_migration.up.sql")
		Expect(err).ToNot(HaveOccurred())
		Expect(upNoTxMigration.Strategy).To(Equal(voyager.SQLNoTransaction))
	})

	Context("SQL migrations", func() {
		It("parses the migration into statements", func() {
			bindata.AssetReturns(multipleStatementMigration, nil)
			migration, err := parser.ParseFileToMigration("1234_create_and_alter_table.up.sql")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(migration.Statements)).To(Equal(2))
		})

		It("combines sql functions in one statement", func() {
			bindata.AssetReturns(sqlFunctionMigration, nil)
			migration, err := parser.ParseFileToMigration("1800_sql_function_migration.up.sql")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(migration.Statements)).To(Equal(1))
		})

		It("removes the BEGIN and COMMIT statements", func() {
			bindata.AssetReturns(multipleStatementMigration, nil)

			migration, err := parser.ParseFileToMigration("1234_create_and_alter_table.up.sql")
			Expect(err).ToNot(HaveOccurred())
			Expect(len(migration.Statements)).To(Equal(2))
			Expect(migration.Statements[0]).ToNot(Equal("BEGIN"))
		})

		Context("No transactions", func() {
			It("marks migration as no transaction", func() {
				bindata.AssetReturns(noTransactionMigration, nil)

				migration, err := parser.ParseFileToMigration("3000_some_no_transaction_migration.up.sql")
				Expect(err).ToNot(HaveOccurred())
				Expect(len(migration.Statements)).To(Equal(1))
			})
		})
	})

	Context("Go migrations", func() {
		It("returns the name of the migration function to run", func() {
			bindata.AssetReturns([]byte(`
				func (m *Migrator) Up_2000() {}
			`), nil)

			migration, err := parser.ParseFileToMigration("2000_some_go_migration.up.go")
			Expect(err).ToNot(HaveOccurred())
			Expect(migration.Name).To(Equal("Up_2000"))
		})
	})

})
