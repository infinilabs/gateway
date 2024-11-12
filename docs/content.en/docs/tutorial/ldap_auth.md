---
title: "LDAP Authentication Integrity"
weight: 10
draft: true
---

# LDAP Authentication Integrity

## Overview of LDAP

The Lightweight Directory Access Protocol (LDAP) is an information service that provides the directory service. The directory service is a special database system that is specifically optimized for reading, browsing, and searching. Directories are typically used to contain descriptive attribute-based information and support sophisticated filtering capabilities.
Directories generally do not support complex transaction management or rollback strategies required by common databases for a large number of update operations. The update of the directory service is generally very simple.
The directories can store personal information, Web links, JPEG images, and other information. To access information stored in directories, you need to use LDAP, which is an access protocol running over TCP/IP.

Information in an LDAP directory is organized in a tree structure. Information is stored in the data structure of entries. Entries are equivalent to records in tables of a relational database. An entry is an attribute with a distinguished name (DN) and the DN is used to reference the entry. A DN is equivalent to the primary key in a table of a relational database. An attribute is composed of the type and one or more values, and is equivalent to a field in a relational database composed of the field name and data type. To facilitate searches, a LDAP type can have multiple values. This is different from a relational database, in which fields must be irrelevant in order to reduce data redundancy. LDAP entries are organized by geographical location and organization relationship, which is very intuitive. LDAP stores data in files. To improve efficiency, you can use an index-based file database rather than a relational database. One type example is mail and the value is an email address.

LDAP information is stored in a tree structure. Countries (c = CN) or domain names (dc = com) are generally defined at the root, and one or more organizations (o = Acme) or organizational units (ou = People) are usually defined below. One organizational unit may contain all employees, all printers in the building, and other information. In addition, LDAP is capable of controlling attributes that can be and must be supported by entries. This is implemented by one special attribute called objectClass. The value of this attribute determines the rules that an entry must conform to, and the rules specify what attributes the entry can and at least should contain. For example, the inetorgPerson object class needs to support the surname (sn) and common name (cn) attributes, but can also contain optional attributes such as email and telephone number.

## LDAP Abbreviations

- o – organization (organization - company)
- ou – organization unit (organization unit - department)
- c - countryName (country)
- dc - domainComponent (domain name)
- sn – surname (real name)
- cn - common name
