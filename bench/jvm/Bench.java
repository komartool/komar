import java.util.ArrayList;
import java.util.List;

public class Bench {
    static final class Customer {
        private final String name;
        private final boolean vip;

        Customer(String name, boolean vip) {
            this.name = name;
            this.vip = vip;
        }
    }

    static final class Order {
        private final long id;
        private final double total;
        private final Customer customer;
        private final List<String> items = new ArrayList<>();

        Order(long id, double total, Customer customer) {
            this.id = id;
            this.total = total;
            this.customer = customer;
            items.add("a");
            items.add("b");
        }
    }

    static long sink;

    static int target(Order order, int qty) {
        int result = (int) order.total * qty;
        return result + 1;
    }

    static int caller(Order order, int qty) {
        return Math.addExact(target(order, qty), 1);
    }

    public static void main(String[] args) {
        int n = Integer.parseInt(args[0]);
        int warmup = n / 5;
        Order order = new Order(42, 1500.0, new Customer("ann", true));
        for (int i = 0; i < warmup; i++) {
            sink += caller(order, i & 7);
        }
        long start = System.nanoTime();
        for (int i = 0; i < n; i++) {
            sink += caller(order, i & 7);
        }
        long elapsed = System.nanoTime() - start;
        System.out.println("ns_per_call " + (double) elapsed / n);
        System.out.println("sink " + sink);
    }
}
