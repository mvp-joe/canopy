require_relative 'store'
require_relative 'validators'

class OrderProcessor
  attr_reader :store

  def initialize
    @store = Store.new
  end

  def create_order(customer_name, items)
    Validators.validate_name(customer_name)
    Validators.validate_items(items)

    total = items.sum { |item| item[:price] * item[:quantity] }
    order = {
      id: @store.next_id,
      customer: customer_name,
      items: items,
      total: total,
      status: :pending
    }
    @store.save(order)
    order
  end

  def process_order(id)
    order = @store.find(id)
    raise "Order not found" unless order
    order[:status] = :processed
    @store.update(order)
    order
  end

  def list_orders(status = nil)
    orders = @store.all
    return orders unless status
    orders.select { |o| o[:status] == status }
  end
end

if __FILE__ == $0
  processor = OrderProcessor.new
  order = processor.create_order("Alice", [
    { name: "Widget", price: 9.99, quantity: 2 },
    { name: "Gadget", price: 24.99, quantity: 1 }
  ])
  puts "Created order ##{order[:id]} for #{order[:customer]}: $#{order[:total]}"
end
