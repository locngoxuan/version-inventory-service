package main

import (
	"context"
	"fmt"
	"time"

	"github.com/locngoxuan/xsql"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

type RepoStats struct {
	Namespace       string    `column:"namespace" json:"namespace"`
	RepoId          string    `column:"repo_id" json:"repo_id"`
	CntRelease      int       `column:"cnt_release" json:"cnt_release"`
	CntDevelopment  int       `column:"cnt_development" json:"cnt_development"`
	CntNightly      int       `column:"cnt_nightly" json:"cnt_nightly"`
	CntPatch        int       `column:"cnt_patch" json:"cnt_patch"`
	LastUpdated     time.Time `column:"created" json:"last_update"`
	LastDevelopment string    `column:"version_value" json:"last_development"`
}

type VersionEntity struct {
	Id        string    `column:"id"`
	Namespace string    `column:"namespace"`
	RepoId    string    `column:"repo_id"`
	Type      string    `column:"version_type"`
	Value     string    `column:"version_value"`
	Status    string    `column:"status"`
	Created   time.Time `column:"created"`
}

const (
	versionRelease     = "release"
	versionNightly     = "nightly"
	versionDevelopment = "development"
	versionPatch       = "patch"

	statusCommitted = "committed"
	statusRollback  = "rollback"
)

var vTyps = map[string]struct{}{
	versionDevelopment: {},
	versionNightly:     {},
	versionRelease:     {},
	versionPatch:       {},
}

func (VersionEntity) TableName() string {
	return "versions"
}

func initializeDatabase(ctx context.Context, driver, dsn string) (err error) {
	err = xsql.Open(xsql.DbOption{
		Driver:       driver,
		DSN:          dsn,
		MaxOpenConns: 5,
		MaxIdleConns: 2,
		MaxIdleTime:  600 * time.Second,
		MaxLifeTime:  600 * time.Second,
		Logger:       logger,
	})

	if err != nil {
		return err
	}

	tx, err := xsql.BeginTxContext(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			logger.Panic(p)
		} else if err != nil {
			_ = tx.Rollback() // err is non-nil; don't change it
		} else {
			err = tx.Commit() // err is nil; if Commit returns error update err
			if err != nil {
				_ = tx.Rollback()
			}
		}
	}()

	switch driver {
	case "postgres", "postgresql":
		_, err = tx.ExecContext(ctx, `create table if not exists versions
		(
		  id             varchar(30) not null unique,
		  created        timestamp with time zone default now(),
		  status         varchar(10) not null default 'new',
		  namespace      varchar(255) not null,
		  repo_id   	 varchar(255) not null,
		  version_type   varchar(20) not null,
		  version_value  varchar(20) not null,
		  primary key (id)
		)`)
		if err != nil {
			break
		}
		_, err = tx.ExecContext(ctx, `create index if not exists version_query 
		on versions using btree(namespace, repo_id, version_type)`)
		if err != nil {
			break
		}
	case "sqlite":
		_, err = tx.ExecContext(ctx, `create table if not exists versions
		(
		  id             text not null unique,
		  created        datetime,
		  status         text not null default 'new',
		  namespace      text not null,
		  repo_id   	 text not null,
		  version_type   text not null,
		  version_value  text not null
		)`)
	default:
		err = fmt.Errorf(`driver %s is not supported`, driver)
	}

	return err
}

func deleteVersion(namespace, repoId, typ string) error {
	_, err := xsql.Delete(xsql.NewStmt(`DELETE FROM versions`).
		AppendSql(`WHERE namespace = :namespace AND repo_id = :repo AND version_type = :type`).
		With(map[string]interface{}{
			"namespace": namespace,
			"repo":      repoId,
			"type":      typ,
		}).
		Get())
	return err
}

func findVersion(namespace, repoId, typ string) (VersionEntity, error) {
	var item VersionEntity
	err := xsql.QueryOne(xsql.NewStmt(`SELECT * FROM versions`).
		AppendSql(`WHERE namespace = :namespace AND repo_id = :repo AND version_type = :type`).
		AppendSql(`AND status = 'committed'`).
		AppendSql(`ORDER BY id DESC LIMIT 1`).
		With(map[string]interface{}{
			"namespace": namespace,
			"repo":      repoId,
			"type":      typ,
		}).
		Get(), &item)
	if err != nil {
		return item, err
	}
	return item, nil
}

func changeStatusToCommitted(txId string) error {
	_, err := xsql.Update(xsql.NewStmt(`UPDATE versions`).
		AppendSql(`SET status='committed'`).
		AppendSql(`WHERE id = :id AND status='new'`).
		With(map[string]interface{}{
			"id": txId,
		}).
		ExpectedResult(1).
		Get())
	return err
}

func closeDb() error {
	return xsql.Close()
}

func findAllRepos() ([]RepoStats, error) {
	var items []RepoStats
	err := xsql.Query(xsql.NewStmt(`SELECT t1.namespace, t1.repo_id,`).
		AppendSql(`t2.cnt_development, t2.cnt_release, t2.cnt_patch, t2.cnt_nightly,`).
		AppendSql(`t3.created, t3.version_value`).
		AppendSql(`FROM (`).
		AppendSql(`select distinct namespace, repo_id from versions`).
		AppendSql(`) AS t1`).
		AppendSql(`JOIN (`).
		AppendSql(`select namespace, repo_id,
		sum(case when version_type = 'development' then 1 else 0 end) as cnt_development,
		sum(case when version_type = 'release' then 1 else 0 end) as cnt_release,
		sum(case when version_type = 'patch' then 1 else 0 end) as cnt_patch,
		sum(case when version_type = 'nightly' then 1 else 0 end) as cnt_nightly
	from versions v
	group by namespace, repo_id`).
		AppendSql(`) AS t2 ON t1.namespace = t2.namespace AND t1.repo_id = t2.repo_id`).
		AppendSql(`JOIN (`).
		AppendSql(`select distinct on (namespace, repo_id) namespace, repo_id, version_value, created
		from versions
		order by namespace, repo_id, id desc`).
		AppendSql(`) AS t3 ON t1.namespace = t3.namespace AND t1.repo_id = t3.repo_id`).
		Get(),
		&items)
	return items, err
}
