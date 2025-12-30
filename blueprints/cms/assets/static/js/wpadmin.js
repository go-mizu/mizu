/**
 * WordPress Admin JavaScript
 * Handles interactive functionality for the admin dashboard
 */

(function() {
    'use strict';

    // Initialize when DOM is ready
    document.addEventListener('DOMContentLoaded', function() {
        initMenuToggle();
        initNotices();
        initPostboxes();
        initBulkActions();
        initScreenOptions();
        initTabNavigation();
        initMediaLibrary();
        initPostEditor();
        initMenuEditor();
        initPermalinkSettings();
        initCheckboxToggle();
        initConfirmActions();
        initSearchFilters();
        initDatePickers();
    });

    /**
     * Mobile menu toggle and collapse button
     */
    function initMenuToggle() {
        var collapseBtn = document.getElementById('collapse-button');
        var adminMenu = document.getElementById('adminmenu');
        var wpcontent = document.getElementById('wpcontent');

        // Collapse button handler
        if (collapseBtn) {
            collapseBtn.addEventListener('click', function(e) {
                e.preventDefault();
                document.body.classList.toggle('folded');

                // Update aria-expanded
                var isExpanded = !document.body.classList.contains('folded');
                this.setAttribute('aria-expanded', isExpanded ? 'true' : 'false');

                // Save state
                localStorage.setItem('adminMenuFolded', document.body.classList.contains('folded'));

                // Set cookie for server-side awareness
                document.cookie = 'folded_menu=' +
                    (document.body.classList.contains('folded') ? '1' : '0') +
                    '; path=/wp-admin; max-age=31536000';
            });
        }

        // Restore menu state
        if (localStorage.getItem('adminMenuFolded') === 'true') {
            document.body.classList.add('folded');
            if (collapseBtn) {
                collapseBtn.setAttribute('aria-expanded', 'false');
            }
        }

        // Submenu hover for folded menu
        if (adminMenu) {
            var menuItems = adminMenu.querySelectorAll('.wp-has-submenu');
            menuItems.forEach(function(item) {
                item.addEventListener('mouseenter', function() {
                    if (document.body.classList.contains('folded')) {
                        this.classList.add('opensub');
                    }
                });
                item.addEventListener('mouseleave', function() {
                    if (document.body.classList.contains('folded')) {
                        this.classList.remove('opensub');
                    }
                });

                // Click toggle for submenus in non-folded state
                var menuLink = item.querySelector('a.wp-has-submenu');
                if (menuLink) {
                    menuLink.addEventListener('click', function(e) {
                        if (!document.body.classList.contains('folded')) {
                            // Allow navigation if there's no submenu or clicking on submenu items
                            var submenu = item.querySelector('.wp-submenu');
                            if (submenu && !e.target.closest('.wp-submenu')) {
                                e.preventDefault();
                                item.classList.toggle('wp-menu-open');
                            }
                        }
                    });
                }
            });
        }

        // Mobile menu toggle
        var mobileToggle = document.querySelector('.menu-toggle');
        if (mobileToggle) {
            mobileToggle.addEventListener('click', function(e) {
                e.preventDefault();
                document.body.classList.toggle('mobile-menu-open');
            });
        }
    }

    /**
     * Dismissible admin notices
     */
    function initNotices() {
        var notices = document.querySelectorAll('.notice.is-dismissible');
        notices.forEach(function(notice) {
            var dismissBtn = notice.querySelector('.notice-dismiss');
            if (dismissBtn) {
                dismissBtn.addEventListener('click', function(e) {
                    e.preventDefault();
                    notice.style.opacity = '0';
                    setTimeout(function() {
                        notice.remove();
                    }, 300);
                });
            }
        });
    }

    /**
     * Collapsible postboxes (metaboxes)
     */
    function initPostboxes() {
        var postboxes = document.querySelectorAll('.postbox');
        postboxes.forEach(function(postbox) {
            var toggleBtn = postbox.querySelector('.handlediv, .hndle');
            if (toggleBtn) {
                toggleBtn.addEventListener('click', function() {
                    postbox.classList.toggle('closed');

                    // Save state
                    var postboxId = postbox.id;
                    if (postboxId) {
                        var closedBoxes = JSON.parse(localStorage.getItem('closedPostboxes') || '[]');
                        if (postbox.classList.contains('closed')) {
                            if (closedBoxes.indexOf(postboxId) === -1) {
                                closedBoxes.push(postboxId);
                            }
                        } else {
                            closedBoxes = closedBoxes.filter(function(id) { return id !== postboxId; });
                        }
                        localStorage.setItem('closedPostboxes', JSON.stringify(closedBoxes));
                    }
                });
            }
        });

        // Restore closed postboxes
        var closedBoxes = JSON.parse(localStorage.getItem('closedPostboxes') || '[]');
        closedBoxes.forEach(function(id) {
            var box = document.getElementById(id);
            if (box) {
                box.classList.add('closed');
            }
        });
    }

    /**
     * Bulk actions handling
     */
    function initBulkActions() {
        var bulkForms = document.querySelectorAll('form[data-bulk-action]');
        bulkForms.forEach(function(form) {
            var selectAll = form.querySelector('.check-column input[type="checkbox"]');
            var checkboxes = form.querySelectorAll('tbody input[type="checkbox"]');
            var bulkSelect = form.querySelector('select[name="action"]');
            var applyBtn = form.querySelector('#doaction');

            // Select all checkbox
            if (selectAll) {
                selectAll.addEventListener('change', function() {
                    checkboxes.forEach(function(cb) {
                        cb.checked = selectAll.checked;
                    });
                    updateBulkActionState();
                });
            }

            // Individual checkboxes
            checkboxes.forEach(function(cb) {
                cb.addEventListener('change', function() {
                    updateBulkActionState();

                    // Update select all state
                    if (selectAll) {
                        var allChecked = Array.from(checkboxes).every(function(c) { return c.checked; });
                        var noneChecked = Array.from(checkboxes).every(function(c) { return !c.checked; });
                        selectAll.checked = allChecked;
                        selectAll.indeterminate = !allChecked && !noneChecked;
                    }
                });
            });

            function updateBulkActionState() {
                var anyChecked = Array.from(checkboxes).some(function(c) { return c.checked; });
                if (applyBtn) {
                    applyBtn.disabled = !anyChecked;
                }
            }

            // Form submission
            if (applyBtn) {
                applyBtn.addEventListener('click', function(e) {
                    if (bulkSelect && bulkSelect.value === '-1') {
                        e.preventDefault();
                        alert('Please select a bulk action.');
                        return;
                    }

                    var selected = Array.from(checkboxes).filter(function(c) { return c.checked; });
                    if (selected.length === 0) {
                        e.preventDefault();
                        alert('Please select at least one item.');
                        return;
                    }

                    // Confirm destructive actions
                    if (bulkSelect && bulkSelect.value === 'delete') {
                        if (!confirm('Are you sure you want to delete the selected items? This action cannot be undone.')) {
                            e.preventDefault();
                        }
                    }
                });
            }
        });
    }

    /**
     * Screen options toggle
     */
    function initScreenOptions() {
        var screenOptionsLink = document.getElementById('show-settings-link');
        var screenOptionsWrap = document.getElementById('screen-options-wrap');

        if (screenOptionsLink && screenOptionsWrap) {
            screenOptionsLink.addEventListener('click', function(e) {
                e.preventDefault();
                screenOptionsWrap.classList.toggle('hidden');
                this.setAttribute('aria-expanded',
                    screenOptionsWrap.classList.contains('hidden') ? 'false' : 'true');
            });
        }

        // Help tab
        var helpLink = document.getElementById('contextual-help-link');
        var helpWrap = document.getElementById('contextual-help-wrap');

        if (helpLink && helpWrap) {
            helpLink.addEventListener('click', function(e) {
                e.preventDefault();
                helpWrap.classList.toggle('hidden');
                this.setAttribute('aria-expanded',
                    helpWrap.classList.contains('hidden') ? 'false' : 'true');
            });
        }
    }

    /**
     * Tab navigation (settings pages)
     */
    function initTabNavigation() {
        var navTabs = document.querySelectorAll('.nav-tab-wrapper .nav-tab');
        navTabs.forEach(function(tab) {
            if (tab.getAttribute('href').startsWith('#')) {
                tab.addEventListener('click', function(e) {
                    e.preventDefault();
                    var targetId = this.getAttribute('href').substring(1);
                    var tabContent = document.getElementById(targetId);

                    // Update active tab
                    navTabs.forEach(function(t) { t.classList.remove('nav-tab-active'); });
                    this.classList.add('nav-tab-active');

                    // Show target content, hide others
                    var allPanels = document.querySelectorAll('.tab-panel');
                    allPanels.forEach(function(panel) { panel.classList.add('hidden'); });
                    if (tabContent) {
                        tabContent.classList.remove('hidden');
                    }
                });
            }
        });
    }

    /**
     * Media library grid/list toggle and selection
     */
    function initMediaLibrary() {
        var modeButtons = document.querySelectorAll('.view-switch a');
        var mediaGrid = document.querySelector('.attachments');

        modeButtons.forEach(function(btn) {
            btn.addEventListener('click', function(e) {
                e.preventDefault();
                modeButtons.forEach(function(b) { b.classList.remove('current'); });
                this.classList.add('current');

                if (mediaGrid) {
                    if (this.classList.contains('view-list')) {
                        mediaGrid.classList.remove('attachments-grid');
                        mediaGrid.classList.add('attachments-list');
                    } else {
                        mediaGrid.classList.add('attachments-grid');
                        mediaGrid.classList.remove('attachments-list');
                    }
                }
            });
        });

        // Media item selection
        var mediaItems = document.querySelectorAll('.attachment');
        mediaItems.forEach(function(item) {
            item.addEventListener('click', function(e) {
                if (e.target.closest('a')) return; // Don't trigger on links

                if (e.ctrlKey || e.metaKey) {
                    // Multi-select
                    this.classList.toggle('selected');
                } else if (e.shiftKey && mediaGrid) {
                    // Range select
                    var items = Array.from(mediaGrid.querySelectorAll('.attachment'));
                    var lastSelected = mediaGrid.querySelector('.attachment.last-selected');
                    if (lastSelected) {
                        var start = items.indexOf(lastSelected);
                        var end = items.indexOf(this);
                        var range = items.slice(Math.min(start, end), Math.max(start, end) + 1);
                        range.forEach(function(i) { i.classList.add('selected'); });
                    }
                } else {
                    // Single select
                    mediaItems.forEach(function(i) {
                        i.classList.remove('selected', 'last-selected');
                    });
                    this.classList.add('selected', 'last-selected');
                }
            });
        });
    }

    /**
     * Post editor functionality
     */
    function initPostEditor() {
        // Slug editing
        var slugBtn = document.querySelector('.edit-slug');
        var slugInput = document.getElementById('new-post-slug');
        var slugDisplay = document.getElementById('editable-post-name');

        if (slugBtn && slugInput && slugDisplay) {
            slugBtn.addEventListener('click', function(e) {
                e.preventDefault();
                slugInput.classList.toggle('hidden');
                slugDisplay.classList.toggle('hidden');
                if (!slugInput.classList.contains('hidden')) {
                    slugInput.focus();
                    slugInput.select();
                }
            });
        }

        // Auto-generate slug from title
        var titleInput = document.getElementById('title');
        if (titleInput && slugInput) {
            titleInput.addEventListener('blur', function() {
                if (!slugInput.value || slugInput.dataset.autoGenerate !== 'false') {
                    var slug = this.value
                        .toLowerCase()
                        .replace(/[^a-z0-9]+/g, '-')
                        .replace(/^-|-$/g, '');
                    slugInput.value = slug;
                    if (slugDisplay) {
                        slugDisplay.textContent = slug;
                    }
                }
            });

            slugInput.addEventListener('input', function() {
                this.dataset.autoGenerate = 'false';
            });
        }

        // Featured image
        var featuredImageBtn = document.querySelector('.set-featured-image');
        var removeFeaturedBtn = document.querySelector('.remove-featured-image');
        var featuredImagePreview = document.querySelector('.featured-image-preview');
        var featuredImageInput = document.getElementById('featured_image_id');

        if (featuredImageBtn) {
            featuredImageBtn.addEventListener('click', function(e) {
                e.preventDefault();
                // In a real implementation, this would open a media modal
                // For now, we'll just show an alert
                alert('Media library would open here to select an image.');
            });
        }

        if (removeFeaturedBtn) {
            removeFeaturedBtn.addEventListener('click', function(e) {
                e.preventDefault();
                if (featuredImageInput) featuredImageInput.value = '';
                if (featuredImagePreview) featuredImagePreview.innerHTML = '';
                this.classList.add('hidden');
                if (featuredImageBtn) featuredImageBtn.classList.remove('hidden');
            });
        }

        // Word count
        var contentArea = document.getElementById('content');
        var wordCount = document.getElementById('word-count');

        if (contentArea && wordCount) {
            function updateWordCount() {
                var text = contentArea.value.trim();
                var words = text ? text.split(/\s+/).length : 0;
                wordCount.textContent = words;
            }

            contentArea.addEventListener('input', updateWordCount);
            updateWordCount();
        }

        // Publish confirmation
        var publishBtn = document.getElementById('publish');
        if (publishBtn) {
            publishBtn.addEventListener('click', function(e) {
                var status = document.querySelector('select[name="post_status"]');
                if (status && status.value === 'publish') {
                    if (!confirm('Are you sure you want to publish this post?')) {
                        e.preventDefault();
                    }
                }
            });
        }
    }

    /**
     * Menu editor (drag and drop, nesting)
     */
    function initMenuEditor() {
        var menuContainer = document.getElementById('menu-to-edit');
        if (!menuContainer) return;

        var menuItems = menuContainer.querySelectorAll('.menu-item');
        var draggedItem = null;

        menuItems.forEach(function(item) {
            item.setAttribute('draggable', 'true');

            item.addEventListener('dragstart', function(e) {
                draggedItem = this;
                this.classList.add('dragging');
                e.dataTransfer.effectAllowed = 'move';
            });

            item.addEventListener('dragend', function() {
                this.classList.remove('dragging');
                draggedItem = null;
                updateMenuDepths();
            });

            item.addEventListener('dragover', function(e) {
                e.preventDefault();
                if (draggedItem && draggedItem !== this) {
                    var rect = this.getBoundingClientRect();
                    var midY = rect.top + rect.height / 2;

                    if (e.clientY < midY) {
                        this.parentNode.insertBefore(draggedItem, this);
                    } else {
                        this.parentNode.insertBefore(draggedItem, this.nextSibling);
                    }
                }
            });

            // Toggle item settings
            var handle = item.querySelector('.menu-item-handle');
            var settings = item.querySelector('.menu-item-settings');
            if (handle && settings) {
                handle.addEventListener('click', function() {
                    settings.classList.toggle('hidden');
                    item.classList.toggle('menu-item-edit-active');
                });
            }

            // Remove item
            var removeBtn = item.querySelector('.item-delete');
            if (removeBtn) {
                removeBtn.addEventListener('click', function(e) {
                    e.preventDefault();
                    if (confirm('Are you sure you want to remove this menu item?')) {
                        item.remove();
                        updateMenuDepths();
                    }
                });
            }

            // Indent/outdent buttons
            var indentBtn = item.querySelector('.item-indent');
            var outdentBtn = item.querySelector('.item-outdent');

            if (indentBtn) {
                indentBtn.addEventListener('click', function(e) {
                    e.preventDefault();
                    var currentDepth = parseInt(item.dataset.depth || '0');
                    if (currentDepth < 10) {
                        item.dataset.depth = currentDepth + 1;
                        updateMenuDepths();
                    }
                });
            }

            if (outdentBtn) {
                outdentBtn.addEventListener('click', function(e) {
                    e.preventDefault();
                    var currentDepth = parseInt(item.dataset.depth || '0');
                    if (currentDepth > 0) {
                        item.dataset.depth = currentDepth - 1;
                        updateMenuDepths();
                    }
                });
            }
        });

        function updateMenuDepths() {
            var items = menuContainer.querySelectorAll('.menu-item');
            items.forEach(function(item, index) {
                var depth = parseInt(item.dataset.depth || '0');
                item.style.marginLeft = (depth * 30) + 'px';

                // Update hidden inputs
                var orderInput = item.querySelector('input[name*="[menu-item-position]"]');
                var depthInput = item.querySelector('input[name*="[menu-item-parent-id]"]');
                if (orderInput) orderInput.value = index;
            });
        }

        // Add new menu items
        var addButtons = document.querySelectorAll('.submit-add-to-menu');
        addButtons.forEach(function(btn) {
            btn.addEventListener('click', function(e) {
                e.preventDefault();
                var container = this.closest('.accordion-section-content');
                var checkboxes = container.querySelectorAll('input[type="checkbox"]:checked');

                checkboxes.forEach(function(cb) {
                    var label = cb.nextElementSibling ? cb.nextElementSibling.textContent : cb.value;
                    addMenuItem(label, cb.value, cb.dataset.type || 'custom');
                    cb.checked = false;
                });
            });
        });

        function addMenuItem(label, value, type) {
            var template = document.getElementById('menu-item-template');
            if (!template) return;

            var newItem = template.content.cloneNode(true);
            var item = newItem.querySelector('.menu-item');

            item.querySelector('.menu-item-title').textContent = label;
            item.querySelector('input[name*="[menu-item-title]"]').value = label;
            item.querySelector('input[name*="[menu-item-object-id]"]').value = value;
            item.querySelector('input[name*="[menu-item-type]"]').value = type;

            menuContainer.appendChild(newItem);
            initMenuEditor(); // Reinitialize for new item
        }
    }

    /**
     * Permalink settings - custom structure radio button
     */
    function initPermalinkSettings() {
        var structureRadios = document.querySelectorAll('input[name="selection"]');
        var customInput = document.getElementById('permalink_structure');
        var customRadio = document.getElementById('custom_selection');

        if (customInput && customRadio) {
            structureRadios.forEach(function(radio) {
                radio.addEventListener('change', function() {
                    if (this.id !== 'custom_selection' && this.value !== 'custom') {
                        customInput.value = this.value;
                    }
                });
            });

            customInput.addEventListener('focus', function() {
                if (customRadio) {
                    customRadio.checked = true;
                }
            });

            customInput.addEventListener('input', function() {
                if (customRadio) {
                    customRadio.checked = true;
                }
            });
        }
    }

    /**
     * Checkbox toggle for dependent fields
     */
    function initCheckboxToggle() {
        var toggleCheckboxes = document.querySelectorAll('[data-toggle-target]');
        toggleCheckboxes.forEach(function(cb) {
            var targetId = cb.dataset.toggleTarget;
            var target = document.getElementById(targetId);

            if (target) {
                function updateState() {
                    target.disabled = !cb.checked;
                    if (!cb.checked) {
                        target.classList.add('disabled');
                    } else {
                        target.classList.remove('disabled');
                    }
                }

                cb.addEventListener('change', updateState);
                updateState();
            }
        });
    }

    /**
     * Confirmation dialogs for destructive actions
     */
    function initConfirmActions() {
        var confirmLinks = document.querySelectorAll('[data-confirm]');
        confirmLinks.forEach(function(link) {
            link.addEventListener('click', function(e) {
                var message = this.dataset.confirm || 'Are you sure?';
                if (!confirm(message)) {
                    e.preventDefault();
                }
            });
        });

        // Trash/Delete links
        var trashLinks = document.querySelectorAll('.submitdelete, .trash a, .delete a');
        trashLinks.forEach(function(link) {
            if (!link.dataset.confirm) {
                link.addEventListener('click', function(e) {
                    if (!confirm('Are you sure you want to delete this item?')) {
                        e.preventDefault();
                    }
                });
            }
        });
    }

    /**
     * Search and filter forms
     */
    function initSearchFilters() {
        // Auto-submit filters on change
        var filterSelects = document.querySelectorAll('.tablenav select[name]');
        filterSelects.forEach(function(select) {
            if (select.dataset.autoSubmit !== 'false') {
                select.addEventListener('change', function() {
                    // Don't auto-submit bulk action selects
                    if (this.name !== 'action' && this.name !== 'action2') {
                        this.closest('form').submit();
                    }
                });
            }
        });

        // Clear search
        var searchInputs = document.querySelectorAll('.search-box input[type="search"]');
        searchInputs.forEach(function(input) {
            // Add clear button functionality
            input.addEventListener('input', function() {
                var clearBtn = this.parentNode.querySelector('.clear-search');
                if (clearBtn) {
                    clearBtn.style.display = this.value ? 'inline-block' : 'none';
                }
            });
        });
    }

    /**
     * Date picker initialization
     */
    function initDatePickers() {
        var dateInputs = document.querySelectorAll('input[type="date"], input.datepicker');
        dateInputs.forEach(function(input) {
            // Modern browsers support type="date", but we can enhance with a fallback
            if (input.type !== 'date') {
                // Add placeholder format hint
                input.placeholder = 'YYYY-MM-DD';

                // Basic validation
                input.addEventListener('blur', function() {
                    var value = this.value;
                    if (value && !/^\d{4}-\d{2}-\d{2}$/.test(value)) {
                        this.classList.add('error');
                    } else {
                        this.classList.remove('error');
                    }
                });
            }
        });
    }

    /**
     * Utility: Debounce function
     */
    function debounce(func, wait) {
        var timeout;
        return function executedFunction() {
            var context = this;
            var args = arguments;
            var later = function() {
                timeout = null;
                func.apply(context, args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    }

    /**
     * Utility: Throttle function
     */
    function throttle(func, limit) {
        var lastFunc;
        var lastRan;
        return function() {
            var context = this;
            var args = arguments;
            if (!lastRan) {
                func.apply(context, args);
                lastRan = Date.now();
            } else {
                clearTimeout(lastFunc);
                lastFunc = setTimeout(function() {
                    if ((Date.now() - lastRan) >= limit) {
                        func.apply(context, args);
                        lastRan = Date.now();
                    }
                }, limit - (Date.now() - lastRan));
            }
        };
    }

    /**
     * Welcome panel dismiss
     */
    function initWelcomePanel() {
        var welcomePanel = document.getElementById('welcome-panel');
        var dismissBtn = welcomePanel ? welcomePanel.querySelector('.welcome-panel-close') : null;

        if (dismissBtn) {
            dismissBtn.addEventListener('click', function(e) {
                e.preventDefault();
                welcomePanel.style.transition = 'opacity 0.3s ease, height 0.3s ease';
                welcomePanel.style.opacity = '0';
                welcomePanel.style.height = welcomePanel.offsetHeight + 'px';

                setTimeout(function() {
                    welcomePanel.style.height = '0';
                    welcomePanel.style.overflow = 'hidden';
                    welcomePanel.style.padding = '0';
                    welcomePanel.style.margin = '0';
                    welcomePanel.style.border = 'none';

                    // Save dismissal state
                    localStorage.setItem('welcomePanelDismissed', 'true');
                }, 300);
            });
        }

        // Check if welcome panel was previously dismissed
        if (welcomePanel && localStorage.getItem('welcomePanelDismissed') === 'true') {
            welcomePanel.style.display = 'none';
        }
    }

    /**
     * Accordion sections (menu editor sidebar)
     */
    function initAccordionSections() {
        var accordionTitles = document.querySelectorAll('.accordion-section-title');

        accordionTitles.forEach(function(title) {
            title.addEventListener('click', function() {
                var section = this.closest('.accordion-section');

                // Close other sections (optional - for exclusive accordion)
                // var container = this.closest('.accordion-container');
                // if (container) {
                //     container.querySelectorAll('.accordion-section.open').forEach(function(s) {
                //         if (s !== section) s.classList.remove('open');
                //     });
                // }

                section.classList.toggle('open');
            });
        });
    }

    /**
     * Select all checkboxes in tables
     */
    function initTableSelectAll() {
        var tables = document.querySelectorAll('.wp-list-table');

        tables.forEach(function(table) {
            var selectAllCheckboxes = table.querySelectorAll('thead .check-column input[type="checkbox"], tfoot .check-column input[type="checkbox"]');
            var rowCheckboxes = table.querySelectorAll('tbody .check-column input[type="checkbox"]');

            selectAllCheckboxes.forEach(function(selectAll) {
                selectAll.addEventListener('change', function() {
                    var isChecked = this.checked;
                    rowCheckboxes.forEach(function(cb) {
                        cb.checked = isChecked;
                    });

                    // Sync other select all checkboxes
                    selectAllCheckboxes.forEach(function(other) {
                        other.checked = isChecked;
                    });
                });
            });

            // Update select all when individual checkboxes change
            rowCheckboxes.forEach(function(cb) {
                cb.addEventListener('change', function() {
                    var allChecked = Array.from(rowCheckboxes).every(function(c) { return c.checked; });
                    var noneChecked = Array.from(rowCheckboxes).every(function(c) { return !c.checked; });

                    selectAllCheckboxes.forEach(function(selectAll) {
                        selectAll.checked = allChecked;
                        selectAll.indeterminate = !allChecked && !noneChecked;
                    });
                });
            });
        });
    }

    /**
     * Category add new (inline form)
     */
    function initCategoryAdd() {
        var addToggle = document.getElementById('category-add-toggle');
        var addForm = document.querySelector('.wp-hidden-children .wp-hidden-child');

        if (addToggle && addForm) {
            addToggle.addEventListener('click', function(e) {
                e.preventDefault();
                addForm.style.display = addForm.style.display === 'none' ? 'block' : 'none';

                if (addForm.style.display === 'block') {
                    var input = addForm.querySelector('input[type="text"]');
                    if (input) input.focus();
                }
            });
        }
    }

    /**
     * Tag input handling
     */
    function initTagInput() {
        var tagInputs = document.querySelectorAll('.tagsdiv');

        tagInputs.forEach(function(tagDiv) {
            var input = tagDiv.querySelector('.newtag');
            var addBtn = tagDiv.querySelector('.tagadd, .button');
            var tagList = tagDiv.querySelector('.tagchecklist');
            var hiddenInput = tagDiv.querySelector('input[type="hidden"]');

            if (!input || !tagList) return;

            function addTag(tagName) {
                if (!tagName.trim()) return;

                var span = document.createElement('span');
                span.className = 'tag-item';

                var removeBtn = document.createElement('button');
                removeBtn.type = 'button';
                removeBtn.className = 'ntdelbutton';
                removeBtn.innerHTML = '<span class="remove-tag-icon" aria-hidden="true"></span><span class="screen-reader-text">Remove tag: ' + tagName + '</span>';
                removeBtn.addEventListener('click', function() {
                    span.remove();
                    updateHiddenInput();
                });

                span.textContent = tagName + ' ';
                span.appendChild(removeBtn);
                tagList.appendChild(span);

                input.value = '';
                updateHiddenInput();
            }

            function updateHiddenInput() {
                if (hiddenInput) {
                    var tags = Array.from(tagList.querySelectorAll('.tag-item')).map(function(span) {
                        return span.textContent.trim().replace(/\s*$/, '');
                    });
                    hiddenInput.value = tags.join(', ');
                }
            }

            if (addBtn) {
                addBtn.addEventListener('click', function(e) {
                    e.preventDefault();
                    addTag(input.value);
                });
            }

            input.addEventListener('keypress', function(e) {
                if (e.key === 'Enter' || e.key === ',') {
                    e.preventDefault();
                    addTag(this.value.replace(/,/g, ''));
                }
            });
        });
    }

    /**
     * Form dirty state tracking
     */
    function initFormDirtyState() {
        var forms = document.querySelectorAll('form.dirty-check');

        forms.forEach(function(form) {
            var initialState = new FormData(form);
            var isDirty = false;

            form.addEventListener('change', function() {
                isDirty = true;
            });

            form.addEventListener('input', function() {
                isDirty = true;
            });

            window.addEventListener('beforeunload', function(e) {
                if (isDirty) {
                    e.preventDefault();
                    e.returnValue = 'You have unsaved changes. Are you sure you want to leave?';
                    return e.returnValue;
                }
            });

            form.addEventListener('submit', function() {
                isDirty = false;
            });
        });
    }

    // Initialize additional functions
    document.addEventListener('DOMContentLoaded', function() {
        initWelcomePanel();
        initAccordionSections();
        initTableSelectAll();
        initCategoryAdd();
        initTagInput();
        initFormDirtyState();
    });

    // Expose utilities globally for other scripts
    window.wpAdmin = {
        debounce: debounce,
        throttle: throttle
    };

})();
