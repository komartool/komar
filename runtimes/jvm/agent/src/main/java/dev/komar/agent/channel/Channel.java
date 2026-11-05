package dev.komar.agent.channel;

import java.io.BufferedWriter;
import java.io.IOException;
import java.io.OutputStreamWriter;
import java.io.Writer;
import java.net.InetSocketAddress;
import java.net.Socket;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.ArrayBlockingQueue;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicLong;

public final class Channel {

    private static final int CONNECT_TIMEOUT_MS = 2000;
    private static final int QUEUE_CAPACITY = 10_000;
    private static final long POLL_MS = 100;
    private static final long CLOSE_WAIT_MS = 2000;

    private final Socket socket;
    private final Writer writer;
    private final BlockingQueue<String> queue = new ArrayBlockingQueue<>(QUEUE_CAPACITY);
    private final AtomicLong dropped = new AtomicLong();
    private final Thread sender;
    private volatile boolean closing;

    private Channel(Socket socket) throws IOException {
        this.socket = socket;
        this.writer = new BufferedWriter(new OutputStreamWriter(socket.getOutputStream(), StandardCharsets.UTF_8));
        this.sender = new Thread(this::drain, "komar-sender");
        this.sender.setDaemon(true);
    }

    public static Channel open(String host, int port) throws IOException {
        Socket socket = new Socket();
        socket.setTcpNoDelay(true);
        socket.connect(new InetSocketAddress(host, port), CONNECT_TIMEOUT_MS);
        Channel channel = new Channel(socket);
        channel.sender.start();
        return channel;
    }

    public void send(String record) {
        if (closing || !queue.offer(record)) {
            dropped.incrementAndGet();
        }
    }

    public long dropped() {
        return dropped.get();
    }

    public void close() {
        closing = true;
        try {
            sender.join(CLOSE_WAIT_MS);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }
        closeSocket();
    }

    private void drain() {
        try {
            while (!closing || !queue.isEmpty()) {
                String record = queue.poll(POLL_MS, TimeUnit.MILLISECONDS);
                if (record != null) {
                    write(record);
                }
            }
            write("{\"kind\":\"stats\",\"dropped\":" + dropped.get() + "}");
            writer.flush();
        } catch (IOException e) {
            closing = true;
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }
    }

    private void write(String record) throws IOException {
        writer.write(record);
        writer.write('\n');
        if (queue.isEmpty()) {
            writer.flush();
        }
    }

    private void closeSocket() {
        try {
            socket.close();
        } catch (IOException e) {
            dropped.addAndGet(queue.size());
        }
    }
}
