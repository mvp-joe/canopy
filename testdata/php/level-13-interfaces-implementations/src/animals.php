<?php

interface Animal {
    public function speak(): string;
}

class Dog implements Animal {
    public function speak(): string {
        return "woof";
    }
}

class Cat implements Animal {
    public function speak(): string {
        return "meow";
    }
}

function makeNoise(): void {
    $d = new Dog();
    echo $d->speak();
}
