<?php

class Database {
    private string $dsn;

    public function __construct(string $dsn) {
        $this->dsn = $dsn;
    }

    public function query(string $sql, array $params = []): array {
        return [];
    }

    public function execute(string $sql, array $params = []): int {
        return 0;
    }

    public function getDsn(): string {
        return $this->dsn;
    }
}
