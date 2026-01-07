import React, { useState, useRef, useEffect, useCallback } from 'react';

export interface MenuItem {
  id: string;
  label: string;
  shortcut?: string;
  action?: () => void;
  disabled?: boolean;
  checked?: boolean;
  divider?: boolean;
  submenu?: MenuItem[];
}

export interface Menu {
  id: string;
  label: string;
  items: MenuItem[];
}

export interface MenuBarProps {
  menus: Menu[];
  onMenuAction?: (menuId: string, itemId: string) => void;
}

export const MenuBar: React.FC<MenuBarProps> = ({ menus, onMenuAction }) => {
  const [activeMenu, setActiveMenu] = useState<string | null>(null);
  const [activeSubmenu, setActiveSubmenu] = useState<string | null>(null);
  const menuBarRef = useRef<HTMLDivElement>(null);
  const dropdownRefs = useRef<Map<string, HTMLDivElement>>(new Map());

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuBarRef.current && !menuBarRef.current.contains(event.target as Node)) {
        setActiveMenu(null);
        setActiveSubmenu(null);
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setActiveMenu(null);
        setActiveSubmenu(null);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, []);

  const handleMenuClick = useCallback((menuId: string) => {
    setActiveMenu(activeMenu === menuId ? null : menuId);
    setActiveSubmenu(null);
  }, [activeMenu]);

  const handleMenuHover = useCallback((menuId: string) => {
    if (activeMenu !== null) {
      setActiveMenu(menuId);
      setActiveSubmenu(null);
    }
  }, [activeMenu]);

  const handleItemClick = useCallback((menuId: string, item: MenuItem) => {
    if (item.disabled || item.submenu) return;

    if (item.action) {
      item.action();
    }
    if (onMenuAction) {
      onMenuAction(menuId, item.id);
    }
    setActiveMenu(null);
    setActiveSubmenu(null);
  }, [onMenuAction]);

  const handleSubmenuHover = useCallback((itemId: string) => {
    setActiveSubmenu(itemId);
  }, []);

  const renderMenuItem = (menuId: string, item: MenuItem, index: number) => {
    if (item.divider) {
      return <div key={`divider-${index}`} className="menu-divider" />;
    }

    const hasSubmenu = item.submenu && item.submenu.length > 0;

    return (
      <div
        key={item.id}
        className={`menu-item ${item.disabled ? 'disabled' : ''} ${activeSubmenu === item.id ? 'active' : ''}`}
        onClick={() => handleItemClick(menuId, item)}
        onMouseEnter={() => hasSubmenu && handleSubmenuHover(item.id)}
      >
        <span className="menu-item-check">
          {item.checked && <CheckIcon />}
        </span>
        <span className="menu-item-label">{item.label}</span>
        {item.shortcut && <span className="menu-item-shortcut">{item.shortcut}</span>}
        {hasSubmenu && <span className="menu-item-arrow"><ArrowRightIcon /></span>}
        {hasSubmenu && activeSubmenu === item.id && (
          <div className="menu-submenu">
            {item.submenu!.map((subItem, subIndex) => renderMenuItem(menuId, subItem, subIndex))}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="menu-bar" ref={menuBarRef}>
      {menus.map((menu) => (
        <div
          key={menu.id}
          className={`menu-trigger ${activeMenu === menu.id ? 'active' : ''}`}
          onClick={() => handleMenuClick(menu.id)}
          onMouseEnter={() => handleMenuHover(menu.id)}
        >
          {menu.label}
          {activeMenu === menu.id && (
            <div
              className="menu-dropdown"
              ref={(el) => {
                if (el) dropdownRefs.current.set(menu.id, el);
              }}
            >
              {menu.items.map((item, index) => renderMenuItem(menu.id, item, index))}
            </div>
          )}
        </div>
      ))}
    </div>
  );
};

const CheckIcon = () => (
  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3">
    <polyline points="20 6 9 17 4 12" />
  </svg>
);

const ArrowRightIcon = () => (
  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <polyline points="9 18 15 12 9 6" />
  </svg>
);

export default MenuBar;
