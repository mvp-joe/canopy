<?php

use App\Models\User;

function main(): void {
    $u = new User();
    echo $u->greet();
}
