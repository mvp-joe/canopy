<?php

require_once 'database.php';
require_once 'models.php';

class Router {
    private array $routes = [];

    public function get(string $path, callable $handler): void {
        $this->routes['GET'][$path] = $handler;
    }

    public function post(string $path, callable $handler): void {
        $this->routes['POST'][$path] = $handler;
    }

    public function dispatch(string $method, string $path): mixed {
        if (isset($this->routes[$method][$path])) {
            return call_user_func($this->routes[$method][$path]);
        }
        return null;
    }
}

function startApp(): Router {
    $db = new Database('sqlite::memory:');
    $repo = new ArticleRepository($db);

    $router = new Router();
    $router->get('/articles', function() use ($repo) {
        return $repo->findAll();
    });
    $router->post('/articles', function() use ($repo) {
        return $repo->create('New Article', 'Content here');
    });

    return $router;
}

$app = startApp();
$articles = $app->dispatch('GET', '/articles');
print_r($articles);
