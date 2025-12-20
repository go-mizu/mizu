import {
  type RouteConfig,
  route,
  layout,
  index,
} from "@react-router/dev/routes";

export default [
  layout("routes/_layout.tsx", [
    index("routes/_index.tsx"),
    route("about", "routes/about.tsx"),
    route("users", "routes/users/_layout.tsx", [
      index("routes/users/_index.tsx"),
      route(":id", "routes/users/$id.tsx"),
    ]),
  ]),
] satisfies RouteConfig;
