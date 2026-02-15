module Shapes
  class Shape
    def initialize(name)
      @name = name
    end

    def describe
      "I am a #{@name}"
    end
  end

  class Circle < Shape
    def initialize(radius)
      super("circle")
      @radius = radius
    end

    def area
      Math::PI * @radius ** 2
    end
  end
end
