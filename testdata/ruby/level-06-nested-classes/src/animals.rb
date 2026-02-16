module Animals
  class Base
    def breathe
      "inhale"
    end
  end

  class Dog < Base
    def speak
      "woof"
    end
  end

  class Cat < Base
    def speak
      "meow"
    end
  end
end
