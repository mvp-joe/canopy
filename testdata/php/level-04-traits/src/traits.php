<?php

trait Loggable {
    public function log(string $message): void {
        echo "[LOG] " . $message . "\n";
    }
}

trait Timestamped {
    public function timestamp(): string {
        return date('Y-m-d H:i:s');
    }
}

class Service {
    use Loggable;
    use Timestamped;

    public function run(): void {
        $this->log("running");
    }
}
