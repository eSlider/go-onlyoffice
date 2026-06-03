Vertrag --- Souveräne On-Premise IT-Infrastruktur

*Entwurf auf Basis der Besprechung vom 13.04.2026*

1\. Gegenstand des Vertrags

Individuelle Entwicklung und Bereitstellung einer souveränen
On-Premise-Lösung für den Auftraggeber.\
\
Alle Daten verbleiben ausschließlich auf der firmeneigenen Infrastruktur
des Auftraggebers --- ohne Nutzung von Drittanbieter-Cloud-Diensten.

2\. Anforderungen (Requirements)

  ------------------------------------------------------------------------------
  **\#**                  **Anforderung**            **Beschreibung**
  ----------------------- -------------------------- ---------------------------
  R1                      **Geschützte               Interne und externe
                          Kommunikation**            Unternehmenskommunikation
                                                     über einen vollständig
                                                     selbst gehosteten,
                                                     verschlüsselten Chat-Dienst
                                                     --- ohne Abhängigkeit von
                                                     Drittanbietern (kein Slack,
                                                     Teams, etc.)

  R2                      **Dokumentenmanagement**   Interner und externer
                                                     Dokumentenumlauf mit
                                                     kollaborativer Bearbeitung
                                                     (OnlyOffice), Versionierung
                                                     und Änderungsprotokoll (wer
                                                     hat was wann geändert)

  R3                      **Backup & Disaster        Automatisiertes Backup
                          Recovery**                 aller Daten (Dokumente,
                                                     Chat-Verläufe) auf einen
                                                     externen Datenträger.
                                                     Rollback auf die letzten
                                                     3--4 Tage. Bei
                                                     Hardware-Ausfall:
                                                     Wiederherstellung auf
                                                     Ersatz-Hardware innerhalb
                                                     von ca. ½ Tag

  R4                      **Benutzerverwaltung**     Zentrale Verwaltung von
                                                     Benutzerkonten,
                                                     Zugriffsrechten und Rollen
                                                     für Chat und
                                                     Dokumentenmanagement
  ------------------------------------------------------------------------------

3\. Lieferumfang (Endprodukte)

Der Auftraggeber erhält folgende betriebsbereite Produkte innerhalb
seines Firmennetzwerks:

1.  **Chat-System** --- Selbst gehostete, verschlüsselte
    Kommunikationsplattform (intern + extern)

2.  **OnlyOffice Workspace** --- Dokumentenmanagement mit
    1:1-Kompatibilität zu MS Office-Formaten, kollaborative Bearbeitung,
    Versionskontrolle

3.  **Backup-System** --- Automatisierte Datensicherung auf externe
    Medien mit Wiederherstellungsfunktion

4.  **Admin-Panel** --- Benutzerverwaltung, Rechtevergabe,
    Systemüberwachung

5.  **Fernwartungszugang** --- Geschützter Kanal für laufenden Support
    (RustDesk mit eigenem Relay-Server)

4\. Phasen und Zeitplan

  ----------------------------------------------------------------------------------------------
  **Phase**         **Bezeichnung**             **Inhalt**                     **Geschätzte
                                                                               Dauer**
  ----------------- --------------------------- ------------------------------ -----------------
  **1**             Hardware-Beschaffung &      Auswahl und Zusammenbau des    3--5 Tage
                    Zusammenbau                 Servers nach Anforderungen des 
                                                Auftraggebers                  

  **2**             Netzwerk &                  Aufstellung vor Ort beim       2 Tage
                    Grundkonfiguration          Auftraggeber. Konfiguration    
                                                von Router/FritzBox,           
                                                Proxy-Manager, DNS,            
                                                geschütztem                    
                                                Remote-Admin-Zugang            
                                                (VPN/RustDesk)                 

  **3**             Chat-Bereitstellung         Installation und Konfiguration 3--5 Tage
                                                der selbst gehosteten          
                                                Chat-Lösung, Anbindung an die  
                                                Benutzerverwaltung             

  **4**             OnlyOffice-Bereitstellung   Installation von OnlyOffice    5--7 Tage
                                                Workspace,                     
                                                WebDAV-Konfiguration,          
                                                Netzlaufwerke,                 
                                                Backup-Einrichtung auf         
                                                externem Datenträger           

  **5**             Replikation &               Einrichtung eines              2--3 Tage
                    Ausfallsicherheit           Backbone-Kanals zur            
                    *(optional)*                gegenseitigen Replikation      
                                                zwischen Standorten            
                                                (Disaster-Recovery-Szenario)   

  **6**             Schulung                    Schulung des internen          1--2 Tage
                                                Personals und des              
                                                Auftraggebers in der Nutzung   
                                                aller Systeme. Durchführung    
                                                remote über den geschützten    
                                                Kommunikationskanal            

                    **Gesamtlaufzeit**                                         **ca. 4 Wochen**
  ----------------------------------------------------------------------------------------------

5\. Kostenaufstellung

  ------------------------------------------------------------------------
  **Pos.**                **Leistung**             **Betrag (netto)**
  ----------------------- ------------------------ -----------------------
  1                       Hardware (Server, ext.   2.000 €
                          Datenträger)             

  2                       Netzwerk- &              7.000 €
                          Grundkonfiguration       
                          (Phase 2)                

  3                       Chat-Entwicklung &       20.000 €
                          Bereitstellung (Phase 3) 

  4                       OnlyOffice-Entwicklung & 20.000 €
                          Bereitstellung (Phase 4) 

  5                       Schulung (Phase 6)       5.000 €

  6                       Vertragsabschluss &      5.000 €
                          Projektmanagement        

                                                   

                          **Zwischensumme**        **59.000 €**

                          zzgl. MwSt. (19 %)       11.210 €

                          **Gesamtbetrag           **\~65.000 €** \*
                          (brutto)**               
  ------------------------------------------------------------------------

*\* Endpreis nach Abstimmung; Diskussionsspanne 55.000--65.000 €
brutto.*

6\. Laufender Support

  -----------------------------------------------------------------------
  **Leistung**                        **Umfang**
  ----------------------------------- -----------------------------------
  Fernwartung                         Über geschützten Kanal (RustDesk
                                      mit eigenem Relay-/ID-Server)

  Reaktionszeit                       Innerhalb eines Arbeitstages

  Fehlerbehebung                      Bug-Fixes, Service-Neustarts,
                                      WebDAV-/Kompatibilitätsprobleme

  Updates                             Regelmäßige Sicherheits- und
                                      Feature-Updates nach Absprache

  Backup-Monitoring                   Überprüfung der Backup-Integrität
  -----------------------------------------------------------------------

*Monatliche Support-Pauschale: **nach Vereinbarung** (in Gesamtsumme
enthalten oder separat).*

7\. Voraussetzungen seitens des Auftraggebers

-   Bereitstellung eines Stellplatzes und Stromanschlusses für den
    Server

-   Internetzugang (FritzBox o. ä.) mit Möglichkeit zur
    Port-Weiterleitung

-   Benennung eines internen Ansprechpartners

-   Teilnahme an der Schulung (Phase 6)

8\. Alleinstellungsmerkmale (USP)

  -----------------------------------------------------------------------
  **Aspekt**                          **Vorteil**
  ----------------------------------- -----------------------------------
  **Datensouveränität**               Alle Daten verbleiben physisch im
                                      Unternehmen --- kein
                                      Cloud-Anbieter, kein US-Recht
                                      (CLOUD Act), volle
                                      DSGVO-Konformität

  **Open Source**                     Basierend auf
                                      Open-Source-Technologien
                                      (OnlyOffice, RustDesk) --- keine
                                      Vendor-Lock-in-Risiken

  **MS-Office-Kompatibilität**        OnlyOffice bietet
                                      1:1-Formatkompatibilität --- im
                                      Gegensatz zu LibreOffice bleibt die
                                      Formatierung erhalten

  **Aktuelle Marktlage**              Frankreich und weitere EU-Länder
                                      stellen auf souveräne Lösungen um;
                                      steigende Budgets für
                                      On-Premise-Infrastruktur

  **Erprobte Lösung**                 Bereits bei mehreren Kunden im
                                      Einsatz (Referenzen verfügbar)
  -----------------------------------------------------------------------

9\. Bekannte technische Hinweise

-   **WebDAV + MS Word:** Word erstellt unsichtbare Lock-Dateien (mit
    Sonderzeichen \~\$, #), die den WebDAV-Dienst stören können → wird
    durch Filterung im Backend-Service behandelt

-   **Hibernation:** Client-PCs mit aktiviertem Ruhezustand verlieren
    die Verbindung zum Netzlaufwerk → Hibernation auf
    Arbeitsplatzrechnern deaktivieren

-   **RustDesk Relay:** Eigener ID-/Relay-Server empfohlen, um
    Abhängigkeit von öffentlichen RustDesk-Servern zu vermeiden und
    stabile Verbindungen zu gewährleisten

10\. Nächste Schritte

-   Kontaktaufnahme mit Michel --- Vertragsentwurf vorlegen

-   Hardware-Spezifikation finalisieren

-   Zeitplan mit Auftraggeber abstimmen

-   Referenzseite auf Website veröffentlichen (3 Referenzen: Spanien + 2
    DE)

-   Videoaufnahmen von zufriedenen Kunden für Marketing

*Quelle: Gesprächsaufzeichnung vom 13.04.2026*
