<?php

require_once 'base.php';

class Circle extends Shape {
    private float $radius;

    public function __construct(float $radius) {
        $this->radius = $radius;
    }

    public function area(): float {
        return 3.14159 * $this->radius * $this->radius;
    }
}

function main(): void {
    $c = new Circle(5.0);
    echo $c->area() . "\n";
    echo $c->describe() . "\n";
}
