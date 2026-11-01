import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'package:provider/provider.dart';

import 'api_config.dart';

Future<Map<String, dynamic>> managementRequest(
    BuildContext context, String method, String path,
    [Map<String, dynamic>? body]) async {
  final request = http.Request(method, Uri.parse('${ApiConfig.baseUrl}$path'))
    ..headers['Content-Type'] = 'application/json';
  if (body != null) request.body = jsonEncode(body);
  final response = await http.Response.fromStream(
      await context.read<http.Client>().send(request));
  if (response.statusCode >= 300) {
    String message = 'Request failed (${response.statusCode})';
    try {
      message = (jsonDecode(response.body)['message'] ?? message).toString();
    } on FormatException {
      // A proxy error may be plain text; keep the HTTP status visible.
    }
    throw http.ClientException(message);
  }
  return jsonDecode(response.body) as Map<String, dynamic>;
}

class ManagementPage extends StatelessWidget {
  const ManagementPage({super.key});

  @override
  Widget build(BuildContext context) => DefaultTabController(
        length: 3,
        child: Scaffold(
          appBar: AppBar(
            title: const Text('System administration'),
            bottom: const TabBar(tabs: [
              Tab(text: 'Users'),
              Tab(text: 'Roles'),
              Tab(text: 'Permissions'),
            ]),
          ),
          body: const TabBarView(children: [
            ManagementResource(kind: 'users'),
            ManagementResource(kind: 'roles'),
            ManagementResource(kind: 'permissions'),
          ]),
        ),
      );
}

class ManagementResource extends StatefulWidget {
  const ManagementResource({super.key, required this.kind});
  final String kind;

  @override
  State<ManagementResource> createState() => _ManagementResourceState();
}

class _ManagementResourceState extends State<ManagementResource> {
  List<Map<String, dynamic>> _rows = [];
  String? _error;
  bool _loading = true;
  int _offset = 0;
  int _nextOffset = 0;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final suffix =
          widget.kind == 'users' ? '?page_size=50&offset=$_offset' : '';
      final result =
          await managementRequest(context, 'GET', '/v1/${widget.kind}$suffix');
      if (!mounted) return;
      setState(() {
        _rows = List<Map<String, dynamic>>.from(result[widget.kind] ?? []);
        _nextOffset = int.tryParse(
                '${result['nextOffset'] ?? result['next_offset'] ?? 0}') ??
            0;
      });
    } catch (error) {
      if (mounted) setState(() => _error = error.toString());
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _edit([Map<String, dynamic>? row]) async {
    final names = [
      'name',
      'description',
      if (widget.kind == 'permissions') ...['resource', 'action']
    ];
    final fields = {
      for (final name in names)
        name: TextEditingController(text: '${row?[name] ?? ''}')
    };
    final accepted = await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
              title: Text('${row == null ? 'Create' : 'Edit'} ${widget.kind}'),
              content: SingleChildScrollView(
                  child: Column(mainAxisSize: MainAxisSize.min, children: [
                for (final name in names)
                  Padding(
                      padding: const EdgeInsets.symmetric(vertical: 6),
                      child: TextField(
                          controller: fields[name],
                          decoration: InputDecoration(labelText: name))),
              ])),
              actions: [
                TextButton(
                    onPressed: () => Navigator.pop(context, false),
                    child: const Text('Cancel')),
                FilledButton(
                    onPressed: () => Navigator.pop(context, true),
                    child: const Text('Save'))
              ],
            ));
    final payload = {
      for (final entry in fields.entries) entry.key: entry.value.text.trim()
    };
    for (final controller in fields.values) {
      controller.dispose();
    }
    if (accepted != true || !mounted) return;
    await _mutate(row == null ? 'POST' : 'PUT',
        '/v1/${widget.kind}${row == null ? '' : '/${row['id']}'}', payload);
  }

  Future<void> _mutate(String method, String path,
      [Map<String, dynamic>? payload]) async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      await managementRequest(context, method, path, payload);
      if (mounted) await _load();
    } catch (error) {
      if (mounted) {
        setState(() {
          _error = error.toString();
          _loading = false;
        });
      }
    }
  }

  Future<void> _remove(Map<String, dynamic> row) async {
    final accepted = await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
              title: Text('Delete ${row['name']}?'),
              content: const Text('Existing assignments may prevent deletion.'),
              actions: [
                TextButton(
                    onPressed: () => Navigator.pop(context, false),
                    child: const Text('Cancel')),
                FilledButton(
                    onPressed: () => Navigator.pop(context, true),
                    child: const Text('Delete'))
              ],
            ));
    if (accepted == true && mounted) {
      await _mutate('DELETE', '/v1/${widget.kind}/${row['id']}');
    }
  }

  @override
  Widget build(BuildContext context) => Column(children: [
        Row(children: [
          IconButton(
              onPressed: _loading ? null : _load,
              tooltip: 'Refresh',
              icon: const Icon(Icons.refresh)),
          if (widget.kind != 'users')
            TextButton.icon(
                onPressed: _loading ? null : _edit,
                icon: const Icon(Icons.add),
                label: const Text('Create'))
        ]),
        if (_loading) const LinearProgressIndicator(),
        if (_error != null)
          Padding(
              padding: const EdgeInsets.all(12),
              child: Text(_error!,
                  style:
                      TextStyle(color: Theme.of(context).colorScheme.error))),
        if (!_loading && _rows.isEmpty && _error == null)
          const Padding(padding: EdgeInsets.all(24), child: Text('No records')),
        Expanded(
            child: ListView.builder(
                itemCount: _rows.length,
                itemBuilder: (context, index) {
                  final row = _rows[index];
                  final reserved =
                      widget.kind == 'roles' && row['name'] == 'super_admin';
                  return ListTile(
                      title: Text('${row['name']}'),
                      subtitle:
                          Text('${row['email'] ?? row['description'] ?? ''}'),
                      trailing: Wrap(
                          children: widget.kind == 'users'
                              ? [
                                  TextButton(
                                      onPressed: _loading
                                          ? null
                                          : () => showDialog<void>(
                                              context: context,
                                              builder: (_) =>
                                                  UserRolesDialog(user: row)),
                                      child: const Text('Roles')),
                                ]
                              : [
                                  if (widget.kind == 'roles')
                                    TextButton(
                                        onPressed: _loading
                                            ? null
                                            : () => showDialog<void>(
                                                context: context,
                                                builder: (_) =>
                                                    RolePermissionsDialog(
                                                        role: row)),
                                        child: const Text('Permissions')),
                                  IconButton(
                                      onPressed: _loading || reserved
                                          ? null
                                          : () => _edit(row),
                                      tooltip: 'Edit',
                                      icon: const Icon(Icons.edit_outlined)),
                                  IconButton(
                                      onPressed: _loading || reserved
                                          ? null
                                          : () => _remove(row),
                                      tooltip: 'Delete',
                                      icon: const Icon(Icons.delete_outline)),
                                ]));
                })),
        if (widget.kind == 'users')
          Row(mainAxisAlignment: MainAxisAlignment.center, children: [
            TextButton(
                onPressed: _loading || _offset == 0
                    ? null
                    : () {
                        _offset = (_offset - 50).clamp(0, _offset);
                        _load();
                      },
                child: const Text('Previous')),
            Text('Page ${_offset ~/ 50 + 1}'),
            TextButton(
                onPressed: _loading || _nextOffset == 0
                    ? null
                    : () {
                        _offset = _nextOffset;
                        _load();
                      },
                child: const Text('Next')),
          ]),
      ]);
}

class UserRolesDialog extends StatefulWidget {
  const UserRolesDialog({super.key, required this.user});
  final Map<String, dynamic> user;
  @override
  State<UserRolesDialog> createState() => _UserRolesDialogState();
}

class _UserRolesDialogState extends State<UserRolesDialog> {
  List<dynamic> _roles = [], _grants = [];
  int? _selected;
  bool _busy = true;
  String? _error;
  final _expiry = TextEditingController();

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _expiry.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final roles = await managementRequest(context, 'GET', '/v1/roles');
      if (!mounted) return;
      final grants = await managementRequest(
          context, 'GET', '/v1/users/${widget.user['id']}/role-details');
      if (mounted) {
        setState(() {
          _roles = roles['roles'] ?? [];
          _grants = grants['userRoles'] ?? grants['user_roles'] ?? [];
          _busy = false;
        });
      }
    } catch (error) {
      if (mounted) {
        setState(() {
          _error = error.toString();
          _busy = false;
        });
      }
    }
  }

  Future<void> _change(int role, bool remove) async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final payload = <String, dynamic>{};
      if (!remove && _expiry.text.trim().isNotEmpty) {
        final raw = _expiry.text.trim();
        if (!RegExp(r'^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$').hasMatch(raw)) {
          throw const FormatException(
              'Use YYYY-MM-DDTHH:MM:SSZ for UTC expiry');
        }
        final time = DateTime.parse(raw).toUtc();
        if (!time.isAfter(DateTime.now().toUtc())) {
          throw const FormatException('Expiry must be in the future');
        }
        payload['expires_at'] = {
          'year': time.year,
          'month': time.month,
          'day': time.day,
          'hours': time.hour,
          'minutes': time.minute,
          'seconds': time.second
        };
      }
      await managementRequest(
          context,
          remove ? 'DELETE' : 'POST',
          '/v1/users/${widget.user['id']}/roles/$role',
          remove ? null : payload);
      if (mounted) await _load();
    } catch (error) {
      if (mounted) {
        setState(() {
          _error = error.toString();
          _busy = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
          title: Text('Roles: ${widget.user['name']}'),
          content: SizedBox(
              width: 500,
              child: SingleChildScrollView(
                  child: Column(mainAxisSize: MainAxisSize.min, children: [
                if (_busy) const LinearProgressIndicator(),
                if (_error != null) Text(_error!),
                for (final grant in _grants)
                  ListTile(
                      title: Text('${grant['role']['name']}'),
                      subtitle: Text(
                          'Expires: ${grant['expiresAt'] ?? grant['expires_at'] ?? 'Never'}'),
                      trailing: IconButton(
                          tooltip: 'Revoke',
                          onPressed: _busy
                              ? null
                              : () => _change(grant['role']['id'] as int, true),
                          icon: const Icon(Icons.remove_circle_outline))),
                DropdownButtonFormField<int>(
                    decoration: const InputDecoration(labelText: 'Role'),
                    items: [
                      for (final role in _roles)
                        DropdownMenuItem(
                            value: role['id'] as int,
                            child: Text('${role['name']}'))
                    ],
                    onChanged: _busy
                        ? null
                        : (value) => setState(() => _selected = value)),
                const SizedBox(height: 12),
                TextField(
                    controller: _expiry,
                    decoration: const InputDecoration(
                        labelText: 'UTC expiry (optional)',
                        hintText: 'YYYY-MM-DDTHH:MM:SSZ')),
                TextButton(
                    onPressed: _busy || _selected == null
                        ? null
                        : () => _change(_selected!, false),
                    child: const Text('Grant role')),
              ]))),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(context),
                child: const Text('Close'))
          ]);
}

class RolePermissionsDialog extends StatefulWidget {
  const RolePermissionsDialog({super.key, required this.role});
  final Map<String, dynamic> role;
  @override
  State<RolePermissionsDialog> createState() => _RolePermissionsDialogState();
}

class _RolePermissionsDialogState extends State<RolePermissionsDialog> {
  List<dynamic> _permissions = [];
  Set<int> _assigned = {};
  bool _busy = true;
  String? _error;
  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final all = await managementRequest(context, 'GET', '/v1/permissions');
      if (!mounted) return;
      final current = await managementRequest(
          context, 'GET', '/v1/roles/${widget.role['id']}/permissions');
      if (mounted) {
        setState(() {
          _permissions = all['permissions'] ?? [];
          _assigned = {
            for (final p in current['permissions'] ?? []) p['id'] as int
          };
          _busy = false;
        });
      }
    } catch (error) {
      if (mounted) {
        setState(() {
          _error = error.toString();
          _busy = false;
        });
      }
    }
  }

  Future<void> _toggle(int id, bool assign) async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await managementRequest(context, assign ? 'POST' : 'DELETE',
          '/v1/roles/${widget.role['id']}/permissions/$id', assign ? {} : null);
      if (mounted) await _load();
    } catch (error) {
      if (mounted) {
        setState(() {
          _error = error.toString();
          _busy = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
          title: Text('Permissions: ${widget.role['name']}'),
          content: SizedBox(
              width: 500,
              height: 400,
              child: Column(children: [
                if (_busy) const LinearProgressIndicator(),
                if (_error != null) Text(_error!),
                Expanded(
                    child: ListView(children: [
                  for (final p in _permissions)
                    CheckboxListTile(
                        title: Text('${p['name']}'),
                        value: _assigned.contains(p['id']),
                        onChanged: _busy || widget.role['name'] == 'super_admin'
                            ? null
                            : (value) => _toggle(p['id'] as int, value!))
                ])),
              ])),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(context),
                child: const Text('Close'))
          ]);
}
