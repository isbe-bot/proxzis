package settlement

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/isbe-bot/bottagon/internal/game/parley"
	"github.com/isbe-bot/bottagon/internal/rating/glicko2"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Never accepts a database URL. Every invocation initializes and owns its own
// cluster, Unix socket, and database; no TCP listener or existing DB is used.
func temporaryPostgres(t *testing.T) *sql.DB {
	t.Helper()
	bin, err := exec.LookPath("initdb")
	if err != nil {
		matches, _ := filepath.Glob("/usr/lib/postgresql/*/bin/initdb")
		if len(matches) == 0 {
			t.Skip("PostgreSQL initdb unavailable; real transaction tests not run")
		}
		bin = matches[len(matches)-1]
	}
	if os.Geteuid() == 0 {
		t.Skip("initdb cannot run as root; run as an unprivileged user")
	}
	dir, err := os.MkdirTemp("", "bottagon-settlement-pg-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Errorf("remove owned cluster: %v", err)
		}
	})
	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(filepath.Join(filepath.Dir(bin), name), args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s: %v\n%s", name, err, out)
		}
	}
	data := filepath.Join(dir, "data")
	run("initdb", "-D", data, "-A", "trust", "-U", "postgres", "--no-locale", "--encoding=UTF8")
	run("pg_ctl", "-D", data, "-l", filepath.Join(dir, "server.log"), "-o", fmt.Sprintf("-F -k %s -h '' -p 5432", dir), "-w", "start")
	t.Cleanup(func() {
		cmd := exec.Command(filepath.Join(filepath.Dir(bin), "pg_ctl"), "-D", data, "-m", "immediate", "-w", "stop")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("stop owned cluster: %v\n%s", err, out)
		}
	})
	db, err := sql.Open("pgx", fmt.Sprintf("host=%s port=5432 user=postgres dbname=postgres sslmode=disable", dir))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(16)
	t.Cleanup(func() { db.Close() })
	_, file, _, _ := runtime.Caller(0)
	provider, err := goose.NewProvider(goose.DialectPostgres, db,
		os.DirFS(filepath.Join(filepath.Dir(file), "../../migrations")), goose.WithDisableGlobalRegistry(true))
	if err != nil {
		t.Fatal(err)
	}
	migrationCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if _, err := provider.Up(migrationCtx); err != nil {
		t.Fatalf("apply actual Goose migrations: %v", err)
	}

	return db
}

func fixture(t *testing.T, db *sql.DB, p1Balance, p2Balance int) (uuid.UUID, uuid.UUID, *parley.Match) {
	t.Helper()
	p1, p2, id := uuid.New(), uuid.New(), uuid.New()
	for i, bot := range []uuid.UUID{p1, p2} {
		balance := p1Balance
		if i == 1 {
			balance = p2Balance
		}
		if _, err := db.Exec("INSERT INTO bots (id,name,marbles) VALUES ($1,$2,$3)", bot, bot.String(), balance); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec("INSERT INTO matches (id,p1_id,p2_id) VALUES ($1,$2,$3)", id, p1, p2); err != nil {
		t.Fatal(err)
	}
	m := parley.NewMatch(id.String(), parley.DefaultConfig())
	m.Status = parley.StatusCompleted
	m.Winner = parley.Player1
	m.Score.P1 = 4
	return p1, p2, m
}

func assertBalance(t *testing.T, db *sql.DB, id uuid.UUID, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow("SELECT marbles FROM bots WHERE id=$1", id).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("bot %s balance=%d want=%d", id, got, want)
	}
}
func assertLedger(t *testing.T, db *sql.DB, id uuid.UUID, wantCount int) {
	t.Helper()
	var count int
	var sum int64
	if err := db.QueryRow("SELECT count(*),coalesce(sum(delta),0) FROM marble_ledger_entries WHERE match_id=$1", id).Scan(&count, &sum); err != nil {
		t.Fatal(err)
	}
	if count != wantCount || sum != 0 {
		t.Errorf("ledger count=%d want=%d sum=%d want=0", count, wantCount, sum)
	}
}

func TestPostgresSettlement(t *testing.T) {
	db := temporaryPostgres(t)
	svc := NewService(db)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	t.Run("unilateral payouts balanced and replay does not change ratings", func(t *testing.T) {
		p1, p2, m := fixture(t, db, 1000, 1000)
		id := uuid.MustParse(m.ID)
		m.Config.P1SideBet = 10
		m.Config.P2SideBet = 5
		m.MirrorTiesCount = 2
		if err := svc.SettleMatch(ctx, id, p1, p2, m); err != nil {
			t.Fatal(err)
		}
		assertBalance(t, db, p1, 1009)
		assertBalance(t, db, p2, 992)
		assertLedger(t, db, id, 4)
		var before, after float64
		if err := db.QueryRow("SELECT rating FROM bots WHERE id=$1", p1).Scan(&before); err != nil {
			t.Fatal(err)
		}
		if err := svc.SettleMatch(ctx, id, p1, p2, m); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow("SELECT rating FROM bots WHERE id=$1", p1).Scan(&after); err != nil {
			t.Fatal(err)
		}
		if before != after {
			t.Errorf("replay changed rating %f -> %f", before, after)
		}
		assertBalance(t, db, p1, 1009)
		assertLedger(t, db, id, 4)
	})
	t.Run("missing match and mismatched participants cannot move money", func(t *testing.T) {
		p1, p2, m := fixture(t, db, 1000, 1000)
		id := uuid.MustParse(m.ID)
		missing := *m
		missing.ID = uuid.NewString()
		if err := svc.SettleMatch(ctx, uuid.MustParse(missing.ID), p1, p2, &missing); !errors.Is(err, ErrMatchNotFound) {
			t.Errorf("missing match error: %v", err)
		}
		if err := svc.SettleMatch(ctx, id, p2, p1, m); err == nil {
			t.Error("accepted swapped participants")
		}
		if err := svc.SettleMatch(ctx, id, p1, p1, m); err == nil {
			t.Error("accepted identical participants")
		}
		assertBalance(t, db, p1, 1000)
		assertBalance(t, db, p2, 1000)
		assertLedger(t, db, id, 0)
	})
	t.Run("concurrent replay settles once", func(t *testing.T) {
		p1, p2, m := fixture(t, db, 1000, 1000)
		id := uuid.MustParse(m.ID)
		var wg sync.WaitGroup
		errs := make(chan error, 12)
		for i := 0; i < 12; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); errs <- svc.SettleMatch(ctx, id, p1, p2, m) }()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Error(err)
			}
		}
		assertBalance(t, db, p1, 1001)
		assertBalance(t, db, p2, 999)
		assertLedger(t, db, id, 4)
	})
	t.Run("insufficient funds rolls back winner ledger and rating", func(t *testing.T) {
		p1, p2, m := fixture(t, db, 1000, 0)
		id := uuid.MustParse(m.ID)
		if err := svc.SettleMatch(ctx, id, p1, p2, m); err == nil {
			t.Fatal("accepted insolvent loser")
		}
		assertBalance(t, db, p1, 1000)
		assertBalance(t, db, p2, 0)
		assertLedger(t, db, id, 0)
		var rating float64
		var status string
		if err := db.QueryRow("SELECT rating FROM bots WHERE id=$1", p1).Scan(&rating); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow("SELECT status FROM matches WHERE id=$1", id).Scan(&status); err != nil {
			t.Fatal(err)
		}
		if rating != 1500 || status != "pending" {
			t.Errorf("failed tx changed rating/status: %f/%s", rating, status)
		}
	})
	t.Run("conflicting replay is rejected", func(t *testing.T) {
		p1, p2, m := fixture(t, db, 1000, 1000)
		id := uuid.MustParse(m.ID)
		if err := svc.SettleMatch(ctx, id, p1, p2, m); err != nil {
			t.Fatal(err)
		}
		for _, change := range []func(*parley.Match){
			func(m *parley.Match) { m.Winner = parley.Player2 },
			func(m *parley.Match) { m.Config.P1SideBet++ },
			func(m *parley.Match) { m.MirrorTiesCount++ },
			func(m *parley.Match) { m.Score.P1++ },
			func(m *parley.Match) { m.IsSuddenDeath = true },
		} {
			copy := *m
			change(&copy)
			if err := svc.SettleMatch(ctx, id, p1, p2, &copy); !errors.Is(err, ErrSettlementConflict) {
				t.Errorf("conflicting replay: %v", err)
			}
		}
		assertBalance(t, db, p1, 1001)
		assertBalance(t, db, p2, 999)
		assertLedger(t, db, id, 4)
	})
	t.Run("completed outcome without financial marker can settle", func(t *testing.T) {
		p1, p2, m := fixture(t, db, 1000, 1000)
		id := uuid.MustParse(m.ID)
		if _, err := db.Exec("UPDATE matches SET status='completed' WHERE id=$1", id); err != nil {
			t.Fatal(err)
		}
		if err := svc.SettleMatch(ctx, id, p1, p2, m); err != nil {
			t.Fatal(err)
		}
		var settled bool
		if err := db.QueryRow("SELECT settled_at IS NOT NULL FROM matches WHERE id=$1", id).Scan(&settled); err != nil {
			t.Fatal(err)
		}
		if !settled {
			t.Fatal("missing durable marker")
		}
		assertLedger(t, db, id, 4)
	})
	t.Run("legacy ledger without marker fails closed", func(t *testing.T) {
		p1, p2, m := fixture(t, db, 1000, 1000)
		id := uuid.MustParse(m.ID)
		if _, err := db.Exec(`INSERT INTO marble_ledger_entries
   (id,bot_id,match_id,account_type,entry_type,delta,balance_after)
   VALUES ($1,$2,$3,'bot','match_win',0,1000)`, uuid.New(), p1, id); err != nil {
			t.Fatal(err)
		}
		if err := svc.SettleMatch(ctx, id, p1, p2, m); !errors.Is(err, ErrSettlementConflict) {
			t.Fatalf("legacy retry: %v", err)
		}
		assertBalance(t, db, p1, 1000)
		assertBalance(t, db, p2, 1000)
		assertLedger(t, db, id, 1)
	})
	t.Run("draw still pays mirror penalties to house", func(t *testing.T) {
		p1, p2, m := fixture(t, db, 1000, 1000)
		id := uuid.MustParse(m.ID)
		m.Winner = ""
		m.Score.P1 = 0
		m.Config.P1SideBet = 10
		m.MirrorTiesCount = 2
		if err := svc.SettleMatch(ctx, id, p1, p2, m); err != nil {
			t.Fatal(err)
		}
		assertBalance(t, db, p1, 998)
		assertBalance(t, db, p2, 998)
		assertLedger(t, db, id, 4)
	})
	t.Run("house contestant can go negative but human cannot", func(t *testing.T) {
		p1, p2, m := fixture(t, db, 1000, 0)
		id := uuid.MustParse(m.ID)
		if _, err := db.Exec("UPDATE bots SET is_house=true WHERE id=$1", p2); err != nil {
			t.Fatal(err)
		}
		if err := svc.SettleMatch(ctx, id, p1, p2, m); err != nil {
			t.Fatal(err)
		}
		assertBalance(t, db, p2, -1)
		assertLedger(t, db, id, 4)
	})
	t.Run("int32 bot upper and lower bounds rollback", func(t *testing.T) {
		for _, upper := range []bool{true, false} {
			b1, b2 := 1000, math.MinInt32
			if upper {
				b1, b2 = math.MaxInt32, 1000
			}
			p1, p2, m := fixture(t, db, b1, b2)
			id := uuid.MustParse(m.ID)
			if _, err := db.Exec("UPDATE bots SET is_house=true WHERE id=$1", p2); err != nil {
				t.Fatal(err)
			}
			if err := svc.SettleMatch(ctx, id, p1, p2, m); !errors.Is(err, ErrBalanceBounds) {
				t.Errorf("bounds rejection: %v", err)
			}
			assertBalance(t, db, p1, b1)
			assertBalance(t, db, p2, b2)
			assertLedger(t, db, id, 0)
		}
	})
	t.Run("late match write failure rolls back all accounts and ratings", func(t *testing.T) {
		p1, p2, m := fixture(t, db, 1000, 1000)
		id := uuid.MustParse(m.ID)
		_, err := db.Exec(fmt.Sprintf(`ALTER TABLE matches ADD CONSTRAINT test_reject_completion
   CHECK (id != '%s'::uuid OR status != 'completed')`, id))
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if _, err := db.Exec("ALTER TABLE matches DROP CONSTRAINT test_reject_completion"); err != nil {
				t.Error(err)
			}
		}()
		var before, after int64
		if err := db.QueryRow("SELECT sum(balance) FROM settlement_house_accounts").Scan(&before); err != nil {
			t.Fatal(err)
		}
		if err := svc.SettleMatch(ctx, id, p1, p2, m); err == nil {
			t.Fatal("expected final UPDATE failure")
		}
		if err := db.QueryRow("SELECT sum(balance) FROM settlement_house_accounts").Scan(&after); err != nil {
			t.Fatal(err)
		}
		if before != after {
			t.Errorf("failed transaction moved house balances %d -> %d", before, after)
		}
		assertBalance(t, db, p1, 1000)
		assertBalance(t, db, p2, 1000)
		assertLedger(t, db, id, 0)
		var rating float64
		if err := db.QueryRow("SELECT rating FROM bots WHERE id=$1", p1).Scan(&rating); err != nil {
			t.Fatal(err)
		}
		if rating != 1500 {
			t.Errorf("failed transaction changed rating to %f", rating)
		}
	})
	t.Run("house bigint bounds rollback", func(t *testing.T) {
		for _, account := range []string{"house_reserve", "house_sink"} {
			var original int64
			if err := db.QueryRow("SELECT balance FROM settlement_house_accounts WHERE account_type=$1", account).Scan(&original); err != nil {
				t.Fatal(err)
			}
			limit := int64(math.MaxInt64)
			if account == "house_reserve" {
				limit = math.MinInt64
			}
			if _, err := db.Exec("UPDATE settlement_house_accounts SET balance=$1 WHERE account_type=$2", limit, account); err != nil {
				t.Fatal(err)
			}
			p1, p2, m := fixture(t, db, 1000, 1000)
			id := uuid.MustParse(m.ID)
			err := svc.SettleMatch(ctx, id, p1, p2, m)
			if _, restoreErr := db.Exec("UPDATE settlement_house_accounts SET balance=$1 WHERE account_type=$2", original, account); restoreErr != nil {
				t.Fatal(restoreErr)
			}
			if !errors.Is(err, ErrBalanceBounds) {
				t.Errorf("house bounds rejection: %v", err)
			}
			assertBalance(t, db, p1, 1000)
			assertBalance(t, db, p2, 1000)
			assertLedger(t, db, id, 0)
		}
	})
	t.Run("different matches with reversed participants serialize without lost updates", func(t *testing.T) {
		p1, p2, m1 := fixture(t, db, 1000, 1000)
		id1 := uuid.MustParse(m1.ID)
		id2 := uuid.New()
		m2 := *m1
		m2.ID = id2.String()
		if _, err := db.Exec("INSERT INTO matches (id,p1_id,p2_id) VALUES ($1,$2,$3)", id2, p2, p1); err != nil {
			t.Fatal(err)
		}
		errs := make(chan error, 2)
		go func() { errs <- svc.SettleMatch(ctx, id1, p1, p2, m1) }()
		go func() { errs <- svc.SettleMatch(ctx, id2, p2, p1, &m2) }()
		for i := 0; i < 2; i++ {
			if err := <-errs; err != nil {
				t.Error(err)
			}
		}
		assertBalance(t, db, p1, 1000)
		assertBalance(t, db, p2, 1000)
		assertLedger(t, db, id1, 4)
		assertLedger(t, db, id2, 4)
	})
	t.Run("match row lock is respected and cancellation is safe", func(t *testing.T) {
		p1, p2, m := fixture(t, db, 1000, 1000)
		id := uuid.MustParse(m.ID)
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if _, err := tx.ExecContext(ctx, "SELECT id FROM matches WHERE id=$1 FOR UPDATE", id); err != nil {
			t.Fatal(err)
		}
		deadline, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
		defer cancel()
		if err := svc.SettleMatch(deadline, id, p1, p2, m); err == nil || deadline.Err() == nil {
			t.Errorf("expected blocked lock cancellation: %v", err)
		}
		if err := tx.Rollback(); err != nil {
			t.Fatal(err)
		}
		assertBalance(t, db, p1, 1000)
		assertLedger(t, db, id, 0)
		if err := svc.SettleMatch(ctx, id, p1, p2, m); err != nil {
			t.Fatalf("retry after cancelled lock: %v", err)
		}
	})
	t.Run("corrupt persisted ratings abort the entire settlement", func(t *testing.T) {
		for _, bad := range []string{"NaN", "Infinity", "-Infinity"} {
			p1, p2, m := fixture(t, db, 1000, 1000)
			id := uuid.MustParse(m.ID)
			if _, err := db.Exec("UPDATE bots SET rating=$1::double precision WHERE id=$2", bad, p2); err != nil {
				t.Fatal(err)
			}
			if err := svc.SettleMatch(ctx, id, p1, p2, m); !errors.Is(err, glicko2.ErrInvalidRating) {
				t.Errorf("corrupt rating should fail: %v", err)
			}
			assertBalance(t, db, p1, 1000)
			assertBalance(t, db, p2, 1000)
			assertLedger(t, db, id, 0)
		}
	})
	t.Run("maximum representable deltas remain balanced", func(t *testing.T) {
		p1, p2, m := fixture(t, db, 0, math.MaxInt32)
		id := uuid.MustParse(m.ID)
		m.Config.P1SideBet = math.MaxInt32 - 1
		m.Config.P2SideBet = math.MaxInt32 - 1
		if err := svc.SettleMatch(ctx, id, p1, p2, m); err != nil {
			t.Fatal(err)
		}
		assertBalance(t, db, p1, math.MaxInt32)
		assertBalance(t, db, p2, 0)
		assertLedger(t, db, id, 4)
	})

	t.Run("player two winner keeps asymmetric payout semantics", func(t *testing.T) {
		p1, p2, m := fixture(t, db, 1000, 1000)
		id := uuid.MustParse(m.ID)
		m.Winner = parley.Player2
		m.Score.P1 = 0
		m.Score.P2 = 4
		m.Config.P1SideBet = 5
		m.Config.P2SideBet = 8
		if err := svc.SettleMatch(ctx, id, p1, p2, m); err != nil {
			t.Fatal(err)
		}
		assertBalance(t, db, p1, 994)
		assertBalance(t, db, p2, 1009)
		assertLedger(t, db, id, 4)
	})
	t.Run("deferred commit failure rolls back financial changes", func(t *testing.T) {
		p1, p2, m := fixture(t, db, 1000, 1000)
		id := uuid.MustParse(m.ID)
		_, err := db.Exec(fmt.Sprintf(`CREATE FUNCTION test_reject_settlement_commit() RETURNS trigger
   LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected commit failure'; END $$;
   CREATE CONSTRAINT TRIGGER test_reject_settlement_commit AFTER UPDATE ON matches
   DEFERRABLE INITIALLY DEFERRED FOR EACH ROW WHEN (NEW.id = '%s'::uuid)
   EXECUTE FUNCTION test_reject_settlement_commit()`, id))
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if _, err := db.Exec("DROP TRIGGER test_reject_settlement_commit ON matches; DROP FUNCTION test_reject_settlement_commit()"); err != nil {
				t.Error(err)
			}
		}()
		if err := svc.SettleMatch(ctx, id, p1, p2, m); err == nil {
			t.Fatal("expected COMMIT failure")
		}
		assertBalance(t, db, p1, 1000)
		assertBalance(t, db, p2, 1000)
		assertLedger(t, db, id, 0)
		var unsettled bool
		if err := db.QueryRow("SELECT settled_at IS NULL FROM matches WHERE id=$1", id).Scan(&unsettled); err != nil {
			t.Fatal(err)
		}
		if !unsettled {
			t.Error("failed commit persisted settlement marker")
		}
	})

	t.Run("house balances reconcile with all counterpart ledger entries", func(t *testing.T) {
		rows, err := db.Query(`SELECT a.account_type,a.balance,coalesce(sum(e.delta),0)
   FROM settlement_house_accounts a LEFT JOIN marble_ledger_entries e USING(account_type)
   GROUP BY a.account_type,a.balance`)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var account string
			var balance, sum int64
			if err := rows.Scan(&account, &balance, &sum); err != nil {
				t.Fatal(err)
			}
			if balance != sum {
				t.Errorf("%s balance=%d ledger sum=%d", account, balance, sum)
			}
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
	})

}
