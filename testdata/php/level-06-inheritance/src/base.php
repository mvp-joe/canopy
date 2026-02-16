<?php

abstract class Shape {
    abstract public function area(): float;

    public function describe(): string {
        return "shape with area " . $this->area();
    }
}
