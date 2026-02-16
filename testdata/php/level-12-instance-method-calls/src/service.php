<?php

class Service {
    public function hello(): string {
        return "hello";
    }

    public function greet(string $name): string {
        return $this->hello() . " " . $name;
    }

    public function run(): void {
        echo $this->greet("world");
    }
}
