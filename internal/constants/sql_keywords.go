package constants

var SQL_KEYWORDS = [...]string{
	"SELECT",
	"FROM",
	"WHERE",
	"JOIN",
	"LEFT",
	"RIGHT",
	"INNER",
	"OUTER",
	"ON",
	"GROUP BY",
	"ORDER BY",
	"HAVING",
	"LIMIT",
	"OFFSET",
	"INSERT",
	"UPDATE",
	"DELETE",
	"VALUES",
	"SET",
	"INTO",
	"CREATE",
	"ALTER",
	"DROP",
	"TABLE",
	"VIEW",
	"INDEX",
	"UNIQUE",
	"PRIMARY",
	"KEY",
	"FOREIGN",
	"REFERENCES",
	"NOT",
	"NULL",
	"DEFAULT",
	"CASCADE",
	"DISTINCT",
	"AND",
	"OR",
	"IN",
	"BETWEEN",
	"LIKE",
	"ASC",
	"DESC",
	"COUNT",
	"SUM",
	"AVG",
	"MIN",
	"MAX",
	"AS",
	"CASE",
	"WHEN",
	"THEN",
	"ELSE",
	"END",
	"EXISTS",
	"ALL",
	"ANY",
	"UNION",
	"INTERSECT",
	"EXCEPT",
	"RETURNING",

	"gen_random_uuid()",
	"NOW()",
	"encode(sha256(((random())::text)::bytea), 'hex'))",
}
