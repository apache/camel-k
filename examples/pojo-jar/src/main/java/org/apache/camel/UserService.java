package org.apache.camel;

import lombok.extern.slf4j.Slf4j;

import java.util.Collection;
import java.util.Map;
import java.util.Random;
import java.util.TreeMap;

@Slf4j
public class UserService {
    private final Map<String, User> users = new TreeMap<>();
    private Random ran = new Random();
    public UserService() {
        users.put("123", new User(123, "John Doe"));
        users.put("456", new User(456, "Donald Duck"));
        users.put("789", new User(789, "Slow Turtle"));
    }

    public User getUser(String id) {
        log.info("getUser", id);
        if ("789".equals(id)) {
            int delay = 500 + ran.nextInt(1500);
            try {
                Thread.sleep(delay);
            } catch (Exception e) {
                // ignore
            }
        }
        return users.get(id);
    }

    public Collection<User> listUsers() {
        log.info("listUsers");
        return users.values();
    }

    public void updateUser(User user) {
        log.info("updateUser", user);
        users.put("" + user.getId(), user);
    }
}
