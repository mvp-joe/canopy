<?php

class Article {
    public int $id;
    public string $title;
    public string $content;
    public string $createdAt;

    public function __construct(int $id, string $title, string $content) {
        $this->id = $id;
        $this->title = $title;
        $this->content = $content;
        $this->createdAt = date('Y-m-d H:i:s');
    }

    public function summary(): string {
        return substr($this->content, 0, 100);
    }
}

class ArticleRepository {
    private Database $db;

    public function __construct(Database $db) {
        $this->db = $db;
    }

    public function findAll(): array {
        return $this->db->query('SELECT * FROM articles');
    }

    public function findById(int $id): ?Article {
        $rows = $this->db->query('SELECT * FROM articles WHERE id = ?', [$id]);
        return empty($rows) ? null : new Article($rows[0]['id'], $rows[0]['title'], $rows[0]['content']);
    }

    public function create(string $title, string $content): Article {
        $this->db->execute('INSERT INTO articles (title, content) VALUES (?, ?)', [$title, $content]);
        return new Article(1, $title, $content);
    }

    public function delete(int $id): bool {
        return $this->db->execute('DELETE FROM articles WHERE id = ?', [$id]) > 0;
    }
}
