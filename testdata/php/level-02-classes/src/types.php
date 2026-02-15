<?php

interface Renderable {
    public function render(): string;
}

class Box implements Renderable {
    private int $width;

    public function __construct(int $width) {
        $this->width = $width;
    }

    public function render(): string {
        return "Box({$this->width})";
    }
}

function createBox(int $w): Box {
    return new Box($w);
}
