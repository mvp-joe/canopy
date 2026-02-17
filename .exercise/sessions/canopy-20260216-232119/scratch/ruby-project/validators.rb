module Validators
  def self.validate_name(name)
    raise ArgumentError, "Name cannot be empty" if name.nil? || name.strip.empty?
    raise ArgumentError, "Name too long" if name.length > 100
  end

  def self.validate_items(items)
    raise ArgumentError, "Items cannot be empty" if items.nil? || items.empty?
    items.each do |item|
      raise ArgumentError, "Item must have name" unless item[:name]
      raise ArgumentError, "Price must be positive" unless item[:price]&.positive?
      raise ArgumentError, "Quantity must be positive" unless item[:quantity]&.positive?
    end
  end

  def self.validate_email(email)
    raise ArgumentError, "Invalid email" unless email =~ /\A[\w+\-.]+@[a-z\d\-]+(\.[a-z\d\-]+)*\.[a-z]+\z/i
  end
end
