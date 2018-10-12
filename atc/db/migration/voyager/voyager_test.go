package voyager_test

import (
	"database/sql"
	"io/ioutil"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/concourse/concourse/atc/db/encryption"
	"github.com/concourse/concourse/atc/db/lock"
	"github.com/concourse/concourse/atc/db/migration/voyager"
	"github.com/concourse/concourse/atc/db/migration/voyager/migrations"
	"github.com/concourse/concourse/atc/db/migration/voyager/runner"
	"github.com/concourse/concourse/atc/db/migration/voyager/voyagerfakes"
	"github.com/lib/pq"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Voyager Migration", func() {
	var (
		err         error
		db          *sql.DB
		lockDB      *sql.DB
		lockFactory lock.LockFactory
		strategy    encryption.Strategy
		source      *voyagerfakes.FakeSource
		runner      runner.MigrationsRunner
	)

	BeforeEach(func() {
		db, err = sql.Open("postgres", postgresRunner.DataSourceName())
		Expect(err).NotTo(HaveOccurred())

		lockDB, err = sql.Open("postgres", postgresRunner.DataSourceName())
		Expect(err).NotTo(HaveOccurred())

		lockFactory = lock.NewLockFactory(lockDB)

		strategy = encryption.NewNoEncryption()
		source = new(voyagerfakes.FakeSource)
		source.AssetStub = asset
		runner = migrations.NewMigrationsRunner(db, strategy)
	})

	AfterEach(func() {
		_ = db.Close()
		_ = lockDB.Close()
	})

	Context("Migration test run", func() {
		It("Runs all the migrations", func() {
			migrator := voyager.NewMigrator(db, lockFactory, strategy, source, runner)

			err := migrator.Up()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Current Version", func() {
		BeforeEach(func() {
			SetupMigrationsHistoryTableToExistAtVersion(db, 2000)
		})

		Context("when the latest migration was an up migration", func() {
			It("reports the current version stored in the database", func() {
				migrator := voyager.NewMigratorForMigrations(db, lockFactory, strategy, source, runner)

				version, err := migrator.CurrentVersion()
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal(2000))
			})
		})

		Context("when the latest migration was a down migration", func() {
			BeforeEach(func() {
				_, err = db.Exec(`INSERT INTO migrations_history(version, tstamp, direction, status, dirty) VALUES($1, current_timestamp, 'up', 'passed', false)`, 3000)
				Expect(err).NotTo(HaveOccurred())
				_, err = db.Exec(`INSERT INTO migrations_history(version, tstamp, direction, status, dirty) VALUES($1, current_timestamp, 'up', 'passed', false)`, 4000)
				Expect(err).NotTo(HaveOccurred())
				_, err = db.Exec(`INSERT INTO migrations_history (version, tstamp, direction, status, dirty) VALUES ($1, current_timestamp, 'down', 'passed', false)`, 4000)
				Expect(err).ToNot(HaveOccurred())
			})

			It("reports the version before the latest down migration", func() {
				migrator := voyager.NewMigratorForMigrations(db, lockFactory, strategy, source, runner)

				version, err := migrator.CurrentVersion()
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal(3000))
			})
		})

		Context("when the latest migration was dirty", func() {
			BeforeEach(func() {
				_, err = db.Exec("INSERT INTO migrations_history (version, tstamp, direction, status, dirty) VALUES (3000, current_timestamp, 'down', 'passed', true)")
				Expect(err).ToNot(HaveOccurred())
			})
			It("throws an error", func() {
				migrator := voyager.NewMigratorForMigrations(db, lockFactory, strategy, source, runner)

				_, err := migrator.CurrentVersion()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Database is in dirty state"))
			})
		})

		Context("when the latest migration failed", func() {
			BeforeEach(func() {
				_, err = db.Exec("INSERT INTO migrations_history (version, tstamp, direction, status, dirty) VALUES (3000, current_timestamp, 'down', 'failed', false)")
				Expect(err).ToNot(HaveOccurred())
			})
			It("reports the version before the failed migration", func() {
				migrator := voyager.NewMigratorForMigrations(db, lockFactory, strategy, source, runner)

				version, err := migrator.CurrentVersion()
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal(2000))
			})
		})
	})

	Context("Supported Version", func() {
		It("SupportedVersion reports the highest supported migration version", func() {
			source.AssetNamesReturns([]string{
				"1000_some_migration.up.sql",
				"3000_this_is_to_prove_we_dont_use_string_sort.up.sql",
				"20000_latest_migration.up.sql",
				"1000_some_migration.down.sql",
				"3000_this_is_to_prove_we_dont_use_string_sort.down.sql",
				"20000_latest_migration.down.sql",
			})
			migrator := voyager.NewMigratorForMigrations(db, lockFactory, strategy, source, runner)

			version, err := migrator.SupportedVersion()
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal(20000))
		})

		It("Ignores files it can't parse", func() {
			source.AssetNamesReturns([]string{
				"1000_some_migration.up.sql",
				"3000_this_is_to_prove_we_dont_use_string_sort.up.sql",
				"20000_latest_migration.up.sql",
				"1000_some_migration.down.sql",
				"3000_this_is_to_prove_we_dont_use_string_sort.down.sql",
				"20000_latest_migration.down.sql",
				"migrations.go",
			})
			migrator := voyager.NewMigratorForMigrations(db, lockFactory, strategy, source, runner)

			version, err := migrator.SupportedVersion()
			Expect(err).NotTo(HaveOccurred())
			Expect(version).To(Equal(20000))
		})
	})

	Context("Upgrade", func() {
		Context("old schema_migrations table exist", func() {
			var dirty bool

			JustBeforeEach(func() {
				SetupSchemaMigrationsTable(db, 8878, dirty)
			})

			Context("dirty state is true", func() {
				BeforeEach(func() {
					dirty = true
				})
				It("errors", func() {

					Expect(err).NotTo(HaveOccurred())

					migrator := voyager.NewMigrator(db, lockFactory, strategy, source, runner)

					err = migrator.Up()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Database is in a dirty state"))

					var newTableCreated bool
					err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='migrations_history')").Scan(&newTableCreated)
					Expect(newTableCreated).To(BeFalse())
				})
			})

			Context("dirty state is false", func() {
				BeforeEach(func() {
					dirty = false
				})

				It("populate migrations_history table with starting version from schema_migrations table", func() {
					startTime := time.Now()
					migrator := voyager.NewMigrator(db, lockFactory, strategy, source, runner)

					err = migrator.Up()
					Expect(err).NotTo(HaveOccurred())

					var (
						version   int
						isDirty   bool
						timeStamp pq.NullTime
						status    string
						direction string
					)
					err = db.QueryRow("SELECT * from migrations_history ORDER BY tstamp ASC LIMIT 1").Scan(&version, &timeStamp, &direction, &status, &isDirty)
					Expect(version).To(Equal(8878))
					Expect(isDirty).To(BeFalse())
					Expect(timeStamp.Time.After(startTime)).To(Equal(true))
					Expect(direction).To(Equal("up"))
					Expect(status).To(Equal("passed"))
				})

				Context("when the migrations_history table already exists", func() {
					It("does not repopulate the migrations_history table", func() {
						SetupMigrationsHistoryTableToExistAtVersion(db, 8878)
						startTime := time.Now()
						migrator := voyager.NewMigrator(db, lockFactory, strategy, source, runner)

						err = migrator.Up()
						Expect(err).NotTo(HaveOccurred())

						var timeStamp pq.NullTime
						rows, err := db.Query("SELECT tstamp FROM migrations_history WHERE version=8878")
						Expect(err).NotTo(HaveOccurred())
						var numRows = 0
						for rows.Next() {
							err = rows.Scan(&timeStamp)
							numRows++
						}
						Expect(numRows).To(Equal(1))
						Expect(timeStamp.Time.Before(startTime)).To(Equal(true))
					})
				})
			})
		})

		Context("sql migrations", func() {
			It("runs a migration", func() {
				simpleMigrationFilename := "1000_test_table_created.up.sql"
				source.AssetReturns([]byte(`
						BEGIN;
						CREATE TABLE some_table (id integer);
						COMMIT;
						`), nil)

				source.AssetNamesReturns([]string{
					simpleMigrationFilename,
				})

				migrator := voyager.NewMigratorForMigrations(db, lockFactory, strategy, source, runner)

				migrations, err := migrator.Migrations()
				Expect(err).NotTo(HaveOccurred())
				Expect(len(migrations)).To(Equal(1))

				err = migrator.Up()
				Expect(err).NotTo(HaveOccurred())

				By("Creating the table in the database")
				var exists string
				err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM information_schema.tables where table_name = 'some_table')").Scan(&exists)
				Expect(err).NotTo(HaveOccurred())
				Expect(exists).To(Equal("true"))

				By("Updating the migrations_history table")
				ExpectDatabaseMigrationVersionToEqual(migrator, 1000)
			})

			It("ignores migrations before the current version", func() {
				SetupMigrationsHistoryTableToExistAtVersion(db, 1000)

				simpleMigrationFilename := "1000_test_table_created.up.sql"
				source.AssetStub = func(name string) ([]byte, error) {
					if name == simpleMigrationFilename {
						return []byte(`
						BEGIN;
						CREATE TABLE some_table (id integer);
						COMMIT;
						`), nil
					}
					return asset(name)
				}
				source.AssetNamesReturns([]string{
					simpleMigrationFilename,
				})

				migrator := voyager.NewMigratorForMigrations(db, lockFactory, strategy, source, runner)
				err := migrator.Up()
				Expect(err).NotTo(HaveOccurred())

				By("Not creating the database referenced in the migration")
				var exists string
				err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM information_schema.tables where table_name = 'some_table')").Scan(&exists)
				Expect(err).NotTo(HaveOccurred())
				Expect(exists).To(Equal("false"))
			})

			It("runs the up migrations in ascending order", func() {
				addTableMigrationFilename := "1000_test_table_created.up.sql"
				removeTableMigrationFilename := "1001_test_table_created.up.sql"

				source.AssetStub = func(name string) ([]byte, error) {
					if name == addTableMigrationFilename {
						return []byte(`
						BEGIN;
						CREATE TABLE some_table (id integer);
						COMMIT;
						`), nil
					} else if name == removeTableMigrationFilename {
						return []byte(`
						BEGIN;
						DROP TABLE some_table;
						COMMIT;
						`), nil
					}
					return asset(name)
				}

				source.AssetNamesReturns([]string{
					removeTableMigrationFilename,
					addTableMigrationFilename,
				})

				migrator := voyager.NewMigratorForMigrations(db, lockFactory, strategy, source, runner)
				err := migrator.Up()
				Expect(err).NotTo(HaveOccurred())

			})

			Context("With a transactional migration", func() {
				It("leaves the database clean after a failure", func() {
					SetupMigrationsHistoryTableToExistAtVersion(db, 1000)
					source.AssetNamesReturns([]string{
						"1200_delete_nonexistent_table.up.sql",
					})

					source.AssetReturns([]byte(`
						DROP TABLE nonexistent;
					`), nil)

					migrator := voyager.NewMigratorForMigrations(db, lockFactory, strategy, source, runner)

					err := migrator.Up()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("rolled back the migration"))
					ExpectDatabaseMigrationVersionToEqual(migrator, 1000)
					ExpectMigrationToHaveFailed(db, 1200, false)
				})
			})

			It("Doesn't fail if there are no migrations to run", func() {
				SetupMigrationsHistoryTableToExistAtVersion(db, 1000)
				source.AssetNamesReturns([]string{
					"1000_initial_migration.up.sql",
				})

				source.AssetReturns([]byte(`
						CREATE table some_table(id int, tstamp timestamp);
				`), nil)

				migrator := voyager.NewMigratorForMigrations(db, lockFactory, strategy, source, runner)
				err = migrator.Up()
				Expect(err).NotTo(HaveOccurred())

				ExpectDatabaseMigrationVersionToEqual(migrator, 1000)

				var someTableExists bool
				err = db.QueryRow(`SELECT EXISTS ( SELECT 1 FROM information_schema.tables WHERE table_name='some_table')`).Scan(&someTableExists)
				Expect(err).ToNot(HaveOccurred())
				Expect(someTableExists).To(Equal(false))
			})

			It("Locks the database so multiple ATCs don't all run migrations at the same time", func() {
				SetupMigrationsHistoryTableToExistAtVersion(db, 900)

				source.AssetNamesReturns([]string{
					"1000_initial_migration.up.sql",
				})

				source.AssetReturns(ioutil.ReadFile("migrations/1000_initial_migration.up.sql"))
				migrator := voyager.NewMigratorForMigrations(db, lockFactory, strategy, source, runner)

				var wg sync.WaitGroup
				wg.Add(3)

				go TryRunUpAndVerifyResult(db, migrator, &wg)
				go TryRunUpAndVerifyResult(db, migrator, &wg)
				go TryRunUpAndVerifyResult(db, migrator, &wg)

				wg.Wait()

				var numRows int
				err := db.QueryRow(`SELECT COUNT(*) from some_table`).Scan(&numRows)
				Expect(err).ToNot(HaveOccurred())
				Expect(numRows).To(Equal(12))
			})

			Context("With a non-transactional migration", func() {
				It("fails if the migration version is in a dirty state", func() {
					source.AssetNamesReturns([]string{
						"1200_delete_nonexistent_table.up.sql",
					})

					source.AssetReturns([]byte(`
							-- NO_TRANSACTION
							DROP TABLE nonexistent;
					`), nil)

					migrator := voyager.NewMigratorForMigrations(db, lockFactory, strategy, source, runner)

					err := migrator.Up()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("Migration.*failed"))

					ExpectMigrationToHaveFailed(db, 1200, true)
				})
			})

		})

		Context("golang migrations", func() {
			It("runs a migration with Migrate", func() {

				migrator := voyager.NewMigratorForMigrations(db, lockFactory, strategy, source, runner)
				source.AssetNamesReturns([]string{
					"1000_initial_migration.up.sql",
					"4000_go_migration.up.go",
				})

				By("applying the initial migration")
				err := migrator.Migrate(1000)
				var columnExists string
				err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM information_schema.columns where table_name = 'some_table' AND column_name = 'name')").Scan(&columnExists)
				Expect(err).NotTo(HaveOccurred())
				Expect(columnExists).To(Equal("false"))

				err = migrator.Migrate(4000)
				Expect(err).NotTo(HaveOccurred())

				By("applying the go migration")
				err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM information_schema.columns where table_name = 'some_table' AND column_name='name')").Scan(&columnExists)
				Expect(err).NotTo(HaveOccurred())
				Expect(columnExists).To(Equal("true"))

				By("updating the migrations history table")
				ExpectDatabaseMigrationVersionToEqual(migrator, 4000)
			})

			It("runs a migration with Up", func() {

				migrator := voyager.NewMigratorForMigrations(db, lockFactory, strategy, source, runner)
				source.AssetNamesReturns([]string{
					"1000_initial_migration.up.sql",
					"4000_go_migration.up.go",
				})

				err := migrator.Up()
				Expect(err).NotTo(HaveOccurred())

				By("applying the migration")
				var columnExists string
				err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM information_schema.columns where table_name = 'some_table' AND column_name = 'name')").Scan(&columnExists)
				Expect(err).NotTo(HaveOccurred())
				Expect(columnExists).To(Equal("true"))

				By("updating the schema migrations table")
				ExpectDatabaseMigrationVersionToEqual(migrator, 4000)
			})
		})
	})

	Context("Downgrade", func() {

		Context("Downgrades to a version with new migrations_history table", func() {
			It("Downgrades to a given version", func() {
				source.AssetNamesReturns([]string{
					"1000_initial_migration.up.sql",
					"2000_update_some_table.up.sql",
					"2000_update_some_table.down.sql",
				})
				migrator := voyager.NewMigratorForMigrations(db, lockFactory, strategy, source, runner)

				err := migrator.Up()
				Expect(err).NotTo(HaveOccurred())

				currentVersion, err := migrator.CurrentVersion()
				Expect(err).NotTo(HaveOccurred())
				Expect(currentVersion).To(Equal(2000))

				err = migrator.Migrate(1000)
				Expect(err).NotTo(HaveOccurred())

				currentVersion, err = migrator.CurrentVersion()
				Expect(err).NotTo(HaveOccurred())
				Expect(currentVersion).To(Equal(1000))

				ExpectToBeAbleToInsertData(db)
			})

			It("Doesn't fail if already at the requested version", func() {
				source.AssetNamesReturns([]string{
					"1000_initial_migration.up.sql",
					"2000_update_some_table.up.sql",
				})
				migrator := voyager.NewMigratorForMigrations(db, lockFactory, strategy, source, runner)

				err := migrator.Migrate(2000)
				Expect(err).NotTo(HaveOccurred())

				currentVersion, err := migrator.CurrentVersion()
				Expect(err).NotTo(HaveOccurred())
				Expect(currentVersion).To(Equal(2000))

				err = migrator.Migrate(2000)
				Expect(err).NotTo(HaveOccurred())

				currentVersion, err = migrator.CurrentVersion()
				Expect(err).NotTo(HaveOccurred())
				Expect(currentVersion).To(Equal(2000))

				ExpectToBeAbleToInsertData(db)
			})

			It("Locks the database so multiple consumers don't run downgrade at the same time", func() {
				migrator := voyager.NewMigratorForMigrations(db, lockFactory, strategy, source, runner)
				source.AssetNamesReturns([]string{
					"1000_initial_migration.up.sql",
					"2000_update_some_table.up.sql",
					"2000_update_some_table.down.sql",
				})

				err := migrator.Up()
				Expect(err).NotTo(HaveOccurred())

				var wg sync.WaitGroup
				wg.Add(3)

				go TryRunMigrateAndVerifyResult(db, migrator, 1000, &wg)
				go TryRunMigrateAndVerifyResult(db, migrator, 1000, &wg)
				go TryRunMigrateAndVerifyResult(db, migrator, 1000, &wg)

				wg.Wait()
			})
		})
	})

})

func TryRunUpAndVerifyResult(db *sql.DB, migrator voyager.Migrator, wg *sync.WaitGroup) {
	defer GinkgoRecover()
	defer wg.Done()

	err := migrator.Up()
	Expect(err).NotTo(HaveOccurred())

	ExpectDatabaseMigrationVersionToEqual(migrator, 1000)
	ExpectToBeAbleToInsertData(db)
}

func TryRunMigrateAndVerifyResult(db *sql.DB, migrator voyager.Migrator, version int, wg *sync.WaitGroup) {
	defer GinkgoRecover()
	defer wg.Done()

	err := migrator.Migrate(version)
	Expect(err).NotTo(HaveOccurred())

	ExpectDatabaseMigrationVersionToEqual(migrator, version)

	ExpectToBeAbleToInsertData(db)
}

func SetupMigrationsHistoryTableToExistAtVersion(db *sql.DB, version int) {
	_, err := db.Exec(`CREATE TABLE migrations_history(version bigint, tstamp timestamp with time zone, direction varchar, status varchar, dirty boolean)`)
	Expect(err).NotTo(HaveOccurred())

	_, err = db.Exec(`INSERT INTO migrations_history(version, tstamp, direction, status, dirty) VALUES($1, current_timestamp, 'up', 'passed', false)`, version)
	Expect(err).NotTo(HaveOccurred())
}

func SetupSchemaMigrationsTable(db *sql.DB, version int, dirty bool) {
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS schema_migrations (version bigint, dirty boolean)")
	Expect(err).NotTo(HaveOccurred())
	_, err = db.Exec("INSERT INTO schema_migrations (version, dirty) VALUES ($1, $2)", version, dirty)
	Expect(err).NotTo(HaveOccurred())
}

func SetupSchemaFromFile(db *sql.DB, path string) {
	migrations, err := ioutil.ReadFile(path)
	Expect(err).NotTo(HaveOccurred())

	for _, migration := range strings.Split(string(migrations), ";") {
		_, err = db.Exec(migration)
		Expect(err).NotTo(HaveOccurred())
	}
}

func ExpectDatabaseMigrationVersionToEqual(migrator voyager.Migrator, expectedVersion int) {
	var dbVersion int
	dbVersion, err := migrator.CurrentVersion()
	Expect(err).NotTo(HaveOccurred())
	Expect(dbVersion).To(Equal(expectedVersion))
}

func ExpectToBeAbleToInsertData(dbConn *sql.DB) {
	rand.Seed(time.Now().UnixNano())

	teamID := rand.Intn(10000)
	_, err := dbConn.Exec("INSERT INTO some_table(id, tstamp) VALUES ($1, current_timestamp)", teamID)
	Expect(err).NotTo(HaveOccurred())
}

func ExpectMigrationToHaveFailed(dbConn *sql.DB, failedVersion int, expectDirty bool) {
	var status string
	var dirty bool
	err := dbConn.QueryRow("SELECT status, dirty FROM migrations_history WHERE version=$1 ORDER BY tstamp desc LIMIT 1", failedVersion).Scan(&status, &dirty)
	Expect(err).NotTo(HaveOccurred())
	Expect(status).To(Equal("failed"))
	Expect(dirty).To(Equal(expectDirty))
}

func ExpectMigrationVersionTableNotToExist(dbConn *sql.DB) {
	var exists string
	err := dbConn.QueryRow("SELECT EXISTS(SELECT 1 FROM information_schema.tables where table_name = 'migration_version')").Scan(&exists)
	Expect(err).NotTo(HaveOccurred())
	Expect(exists).To(Equal("false"))
}

func ExpectDatabaseVersionToEqual(db *sql.DB, version int, table string) {
	var dbVersion int
	query := "SELECT version from " + table + " LIMIT 1"
	err := db.QueryRow(query).Scan(&dbVersion)
	Expect(err).NotTo(HaveOccurred())
	Expect(dbVersion).To(Equal(version))
}
