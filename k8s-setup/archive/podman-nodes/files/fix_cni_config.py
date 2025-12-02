#!/usr/bin/env python3
import json
import sys
import os

def fix_cni_config(config_file):
    """Fix CNI config to use version 0.4.0 and remove problematic plugins"""
    if not os.path.exists(config_file):
        print(f"Config file {config_file} not found")
        return False
    
    try:
        with open(config_file, 'r') as f:
            config = json.load(f)
        
        # Change CNI version to 0.4.0 for firewall plugin compatibility
        config['cniVersion'] = '0.4.0'
        
        # Remove dnsname plugin if present (can cause issues)
        # Also remove any empty plugin objects
        config['plugins'] = [p for p in config['plugins'] if p.get('type') not in ['dnsname', None] and p]
        
        # Write back
        with open(config_file, 'w') as f:
            json.dump(config, f, indent=3)
        
        print(f"Fixed CNI config: {config_file}")
        return True
    except Exception as e:
        print(f"Error fixing CNI config: {e}")
        return False

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Usage: fix_cni_config.py <config_file>")
        sys.exit(1)
    
    success = fix_cni_config(sys.argv[1])
    sys.exit(0 if success else 1)

