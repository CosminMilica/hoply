FROM quay.io/keycloak/keycloak:26.6 AS builder

# Hoply's sign-in theme (themes/hoply); the generator's VS Code theme stays in the tree, unused and not baked.
COPY themes/hoply /opt/keycloak/themes/hoply

# Pre-build optimized Keycloak (caches themes, compiles templates)
RUN /opt/keycloak/bin/kc.sh build

FROM quay.io/keycloak/keycloak:26.6

COPY --from=builder /opt/keycloak/ /opt/keycloak/

ENTRYPOINT ["/opt/keycloak/bin/kc.sh"]
