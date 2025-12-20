import { Outlet } from "react-router";

export default function UsersLayout() {
  return (
    <div className="users-layout">
      <div className="users-header">
        <h1>Users</h1>
      </div>
      <Outlet />
    </div>
  );
}
