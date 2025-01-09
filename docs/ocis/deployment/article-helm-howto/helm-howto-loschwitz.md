Dachzeile: oCIS in Kubernetes mit Helm betreiben
Titel: Gut verwaltet, gut gespeichert

Vorspann: ownCloud Infinite Scale hat die beliebte Lösung für private Datenspeicher nach Cloud-Prinzip auf eine neue Grundlage gestellt und den Anforderungen moderner Umgebungen angepasst. Wer oCIS in Kubernetes betreiben möchte, bekommt dafür nun Schützenhilfe vom Hersteller in Form von einer Helm-Integration.

Autor: Martin Loschwitz

ownCloud ist eine der beliebtesten Lösungen am Markt für das dynamische Speichern vom Daten im Netz. Wer im Sinne der digitalen Souveränität seine kostbaren Informationen nicht auf AWS & Co. ablegen möchte, findet hier eine perfekte Alternative, die sich auf eigener Infrastruktur gut betreiben lässt. Gerade in jüngerer Zeit hat sich bei ownCloud zudem viel getan: Wer beim Begriff ownCloud noch immer an eine auf PHP basierte Lösung denkt, ist beispielsweise schief gewickelt. Stattdessen hat die gleichnamige Firma längst eine neue Version der eigenen Lösung auf den Markt gebracht, die nicht weniger als ein kompletter Rewrite ist. ownCloud Infinite Scale (Abbildung 1), so der Name der runderneuerten Lösung, kommt in Go daher und orientiert sich strikt an den Grundpfeilern der Mikrodienstarchitektur. 

Unter der Haube besteht "oCIS", so die gängige Abkürzung, also aus etlichen kleinen Diensten, von denen jeder eine spezifische Aufgabe beim Speichern von Daten abwickelt. Entsprechend ist auch das typische oCIS-Deploymentszenario ein anderes als beim konventionellen Vorgänger. Wer oCIS betreibt, installiert dieses üblicherweise nicht mehr direkt auf Servern, sondern nutzt die eigens vom Anbieter bereitgestellten Container im Docker-Format und bindet die eigenen Daten per Bind-Mount ein. Das weckt freilich Begehrlichkeiten: Wer sich das händische Ausrollen des Containers sparen möchte, wünscht sich stattdessen den oCIS-Betrieb unter der Ägide eines Flottenmanagers. Die Rolle des Flottenmanagers für Container ist in der IT der Gegenwart Kubernetes fix zugewiesen. Entsprechend möchten viele Admins ein neu zu installierendes oCIS aus Kubernetes heraus betreiben. Die gute Nachricht ist: Dank umfassender Kubernetes-Integration von ownCloud selbst ist der Betrieb von oCIS in Kubernetes heute keine allzu große Herausforderung mehr. Im Zentrum stehen dabei so genannte Helm Charts, also fertige Anweisungen für den Kubernetes-Paketmanager Helm, die ownCloud selbst anbietet und Administratoren ihre Arbeit dadurch massiv erleichtert. Dieser Artikel zeigt am praktischen Beispiel der Kubernetes-Distribution K3s und einer aktuellen oCIS-Version, wie die Installation von oCIS selbst und den begleitenden Komponenten zu bewerkstelligen ist. 

Zuvor steht allerdings ein bisschen Wissenstransfer auf dem Programm, was die grundsätzlichen Details von Kubernetes selbst angeht. Denn K3s ist zwar eine verhältnismäßig einfache K8s-Distribution (K8s ist die gängige Abkürzung für Kubernetes). Es gibt aber auch hier Voraussetzungen, die zwingend erfüllt sein müssen, soll das beschriebene Setup irgendwie sinnvoll auf andere Kubernetes-Umgebungen übertragbar sein. Dabei ist zweitrangig, ob es sich um lokale Kubernetes-Setups handelt, zum Bespiel auf Basis von OpenShift oder Rancher, oder ob fertige K8s-Umgebungen in der Cloud wie EKS, AKS oder GKS zum Einsatz kommen. Bei diesen greifen zusätzlich diverse Automatismen, die zwar toll sind, wenn sie funktionieren -- die aber ein schier undurchdringbares Dickicht bilden, läuft mal etwas schief. Will der Administrator also wirklich wissen, was er tut, statt nur Befehle vertrauensvoll in die Kommandozeile einzugeben, lädt er sich vorher ein ein Stück weit Kubernetes-Wissen auf die Platte.

Grundsätzliches

Kubernetes entsprang einst Googles Feder und hat mittlerweile einen beeindruckenden Werdegang hingelegt. In der IT der Gegenwart gilt es als Königslösung, um beliebige Container und beliebig viele davon über eine Flotte generischer Compute-Knoten zu verteilen. Verteilte Systeme gelten in der IT allerdings als Königsdisziplin. Diese sinnvoll zu lösen erzwingt auf technischer Ebene einiges an Komplexität. Entsprechend ist auch Kubernetes heute hochkomplex. Das liegt vor allem an den verschiedenen Dienst- und Programmschichten, die Kubernetes so abstrahieren muss, dass sie in von Kubernetes verwalteten Containern sinnvoll nutzbar werden. In den klassischen Containerumgebungen der Gegenwart ist alles virtuell: Virtuelles Netz, virtueller Speicher und auch die eigentliche Anwendung im Container selbst. Die Anwendungen, die in Form von Mikrokomponenten für das Deployment in Kubernetes daherkommen, fügen der Gleichung nochmals etliches an eigener Komplexität hinzu. Das geht schon beim Deployment los: Wenn eine Anwendung nicht nur irgendwo einfach laufen muss, sondern wenn viele Anwendungen so laufen müssen, dass sie miteinander kommunizieren können und noch weitere Anforderungen zu erfüllen sind, wäre das händisch ein extremer Aufwand. Eben hier kommt das schon erwähnte Werkzeug Helm ins Spiel, das zwar einen funktionierenden Kubernetes-Cluster bedingt, sich um den größten Teil der Deployment-Herausforderungen dann aber autark  kümmert. Von diesen ist oCIS keineswegs ausgenommen: Es hat schließlich zu den genannten Schnittstellen unmittelbaren Bezug. Ein ownCloud ohne funktionierendes Netz und ohne funktionierenden Speicher hilft nicht.

Und schon die Anforderung des "funktionierenden Kubernetes-Cluster" aus dem vorherigen Absatz ist vor dem Hintergrund eines Artikels wie diesem durchaus eine Herausforderung. Gerade weil Kubernetes so komplex und so vielseitig ist, existieren neben der eigentlichen "Vanilla"-Variante direkt vom Anbieter mittlerweile zahllose Kubernetes-Distributionen. Die sind im Regelfall mit den definierten Standards der Kubernetes-API kompatibel, weisen unter der Haube jedoch enorme Unterschiede zueinander auf. Um Faktoren wie das lokale Netz, die korrekte Konfiguration von DNS und das Ausstellen von SSL-Zertifikaten kümmern sich bei den Hyperscalern AWS, Azure und GCP beispielsweise eigene Dienste, die freilich nativ mit den dortigen, eigenen Kubernetes-Distributionen der Plattform kompatibel sind. Anders ist die Sache gelagert, kommt stattdessen ein lokales OpenShift oder ein lokales Rancher zum Einsatz. Der Vorteil an Kubernetes ist, dass es etwaige Ressourcen selbst so stark abstrahieren kann, dass dieselben Anweisungen des Administrators auf praktisch jeder Kubernetes-Umgebung erfolgreich zu nutzen sind, auch wenn unter der Haube jeweils ganz unterschiedliche Dinge passieren. Die benötigte Abstraktion muss der Administrator, der "nur" oCIS in Kubernetes betreiben möchte, allerdings stets mitdenken. 

Im Falle von oCIS genügt es daher nicht, lediglich oCIS selbst auszurollen. Nötig sind stattdessen auch ein Kontrollmechanismus für eingehende Verbindungen (im Kubernetes-Sprech auch als "Ingress" bezeichnet) sowie ein Management-Werkzeug für DNS-Einträge, das ebenfalls dynamisch direkt aus Kubernetes heraus gesteuert werden kann. Schließlich spielt das Thema SSL eine wichtige Rolle: Eine nicht per SSL-Verbindung abgesicherte oCIS-Instanz wäre weitgehend nutzlos. Damit beim Start von oCIS in einer Kubernetes-Umgebung allerdings ein passendes Zertifikat vorhanden ist, muss das Deployment dieses über einen dynamischen SSL-Dienst wie Let's Encrypt zunächst beschaffen. Die drei genannten Faktoren des eingehenden Traffics, des zu beschaffenden SSL-Zertifikats und des benötigten DNS-Eintrages müssen zudem ineinander greifen. Niemandem hilft schließlich ein SSL-Zertifikat, zu dem der DNS-Eintrag fehlt.

Der folgende Artikel beschreibt das oCIS-Setup mit Helm auf Grundlage einer lokalen K3s-Instanz. Die ist schnell erstellt und leicht zu betreiben. Unmittelbar übertragbar sind die Erkenntnisse aus diesem Artikel durch den Einsatz generischer Komponenten aber grundsätzlich auch auf jede andere Kubernetes-Instanz.

Los geht's

Gegeben sei also ein K3s-Cluster aus drei Compute-Knoten und der üblichen Kubernetes-Management-Ebene mit Cluster Manager, Kubernetes-API und dem Kubernetes-Scheduler. Wie bei K3s üblich ist das Netz mit Flannell konfiguriert, zusätzliche Dienste wie Istio oder Prometheus sind ab Werk zunächst nicht vorhanden. Das Ziel: Mittels des Paketmanagers Helm soll in dieser Kubernetes-Instanz ein ownCloud Infinite Scale mit allen nötigen Zusatzkomponenten laufen, das zudem über die K8s-API steuerbar ist.           
Kubernetes nutzt eine so genannte deklarative Konfiguration. Der Administrator teilt dem Cluster also nicht im Detail mit, welche Arbeitsschritte er zu erledigen hat. Stattdessen übermittelt der Administrator Kubernetes stets eine in YAML oder JSON verfasste Beschreibung des Zustandes, das der Administrator mittels seiner Ressourcen in Kubernetes erreichen möchte. Wie der Dienst dieses Setup dann herstellt, bleibt ihm überlassen. Helm greift in dieses Konzept nahtlos ein: Es produziert dynamisch selbst die Beschreibungen des gewünschten Zustandes und installiert sie dann automatisch in Kubernetes. Der Administrator benötigt deshalb zunächst "helm" als Kommandozeilenwerkzeug selbst. Durchaus untypisch für Lösungen aus dem Kubernetes-Kontext läuft dieses nicht als Programm in K8s, sondern kommt als lokale Binärdatei daher. 

Deren Installation ist allerdings trivial. Von [1] lädt der Administrator sich zunächst einen Tarball der gewünschten Helm-Version herunter. Sprechen keine bekannten Gründe dagegen, ist es sinnvoll, auf die aktuellste Helm-Version zu setzen. Den Tarball entpackt der Administrator anschließend nach "/usr/local/bin/helm". Das Kommando "chmod +x /usr/local/bin/helm" sorgt dafür, dass die Datei auch ausführbar ist. Das war's schon -- wer auf das Hantieren mit direkt aus dem Netz heruntergeladenen Dateien nicht steht, installiert "helm" alternativ über Tools wie Homebrew.

Kubernetes vorbereiten

Damit oCIS in Kubernetes wie beschrieben funktioniert, sind drei zusätzliche Komponenten nötig: Ein Controller für eingehenden Traffic, ein Zertifikatsmanager und eine Lösung, um DNS-Einträge in externen DNS-Servern zu manipulieren. Beim Controller für eingehenden Traffic macht oCIS selbst klare Vorgaben: Die Helm-Charts seiner Autoren nutzen "nginx-ingress", das für eingehende Verbindungen auf der Kubernetes-Ebene eine Nginx-Instanz als Reverse Proxy ("umgekehrter Proxy") ausrollt (Abbildung 2). Für das dynamische Ausstellen von Zertifikaten in Kubernetes hat sich der "cert-manager" etabliert, der mit ACME-kompatiblen Diensten wie Let's Encrypt direkt kommunizieren kann (Abbildung 3). Auch für die Manipulation von Einträgen in DNS-Servern hat sich ein Quasi-Standard etabliert, nämlich das ExternalDNS-Framework, das sogar unter den Fittichen einer offiziellen Special Interest Group ("SIG") innerhalb des Kubernetes-Projektes steht. Um oCIS zu nutzen, installiert der Administrator also zunächst die drei genannten Komponenten in seinen Kubernetes-Cluster. Auch dafür kommt -- zumindest teilweise -- schon der Kubernetes-Paketmanager Helm zum Einsatz. Dieses setzt im Hintergrund allerdings auch auf das Kubernetes-Kontrollwerkzeug "kubectl".

Damit die folgenden Schritte zur Helm-Installation klappen, benötigt der Admin also einerseits "kubectl" selbst und andererseits für dieses eine Konfiguration, die auf den Kubernetes-Cluster zeigt, in dem Helm und letztlich oCIS laufen sollen. Ob "kubectl" eingerichtet ist und funktioniert, prüft das Kommando "kubectl get pods --all-namespaces". Erscheint hiernach eine Liste aller laufenden Container im Cluster, ist "kubectl" einsatzbereit für die folgenden Schritte.

Helm funktioniert im Grunde nach einem simplen Prinzip. Der Administrator hinterlegt in Helm ähnlich wie bei klassischen Paketmanagern so genannte "Repositories". Das sind Verzeichnisse mit Metadaten im Helm-Format, aus denen Helm Paketinformationen extrahiert und anschließend so verarbeitet, dass ein Kubernetes-Cluster daraus die richtigen Ressourcen erzeugt. Entsprechend umfassen die Metadaten in einem "Helm Chart" -- so lautet der Fachbegriff für die Installationsanweisungen, die spezifisch für ein bestimmtes Werkzeug gelten -- Informationen wie die zu startenden Container-Abbilder und etwaige Konfigurationsanweisungen für die Anwendung selbst. 

Die erste im Beispiel zu installierende Komponente für oCIS in Kubernetes ist der Ingress-Controller auf Basis von Nginx. Dazu genügt das Kommando

helm upgrade --install ingress-nginx ingress-nginx \
  --repo https://kubernetes.github.io/ingress-nginx \
  --namespace ingress-nginx --create-namespace
  
Dieses lädt die benötigten Container aus dem Internet in den Cluster und spielt anschließend auch die benötigten Kubernetes-Ressourcen über dessen API in die Umgebung ein. Zeigt im Anschluss der Aufruf von  "kubectl get namespaces" einen Namespace namens "ingress-nginx" an und das Kommando "kubectl get pods ingress-nginx" darin mehrere laufende Pods, hat die Installation funktioniert.

Cert-Manager und DNS

Danach steht die Installation von Cert-Manager an, der für die künftige oCIS-Instanz automatisch ein SSL-Zertifikat über Let's Encrypt besorgt. Für dieses integriert der Administrator zunächst das Verzeichnis, in dem "helm" seine benötigten Metadaten findet:

helm repo add jetstack https://charts.jetstack.io

Anschließend erfolgt die Installation von "cert-manager" wieder mittels des "helm"-Kommandos:

helm install \
  cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --version v1.13.2 \
  --set installCRDs=true

Auch hier lässt sich mittels "kubectl get pods cert-manager" im Anschluss feststellen, ob alles wie gewünscht geklappt hat. Allerdings sollte der Administrator nach "helm" etwas Zeit verstreichen lassen, bevor er nachsieht. Je nach Anbindung des Kubernetes-Clusters kann es nämlich durchaus etwas dauern, bis die Container aus dem Netz heruntergeladen sind und der Dienst fertig ausgerollt ist.

Danach fehlt nur noch die Komponente, die dynamisch DNS-Einträge in externen Nameservern anlegt. Auch hier gibt es eine eigene SIG auf der Kubernetes-Ebene, die die Erweiterung "external-dns" entwickelt und für genau diesen Zweck zur Verfügung stellt. Hier besteht allerdings die Herausforderung, dass der Administrator sich um die ordnungsgemäße Kommunikation zwischen DNS-Server einerseits und seiner Kubernetes-Erweiterung andererseits selbst kümmern muss. Das geht nicht anders: Schließlich wissen die Entwickler der Erweiterung nicht, welcher DNS-Dienst in einem Setup vor Ort in Verwendung ist. In "external-dns" ist grundsätzlich Unterstützung etwa für die DNS-as-a-Service-Dienste der Hyperscaler ebenso enthalten wie Unterstützung für OpenStack Designate. Auch wer einen eigenen DNS-Server etwa mit PowerDNS betreibt, befüttert diesen auf Wunsch -- wie im folgenden Beispiel gezeigt -- aber aus der "external-dns"-Kubernetes-Erweiterung heraus. Wichtig: Die für die Verbindung zu einem Provider benötigten Parameter gibt der Administrator bereits beim Ausrollen des Dienstes mittels Helm an. Zuvor steht aber noch das Hinzufügen des benötigten Helm-Verzeichnisses an:

helm repo add external-dns https://kubernetes-sigs.github.io/external-dns/

Ein entsprechendes Beispiel für die gleichzeitige Installation und Konfiguration von ExternalDNS ist im Anschluss dieses:

helm upgrade --install external-dns external-dns/external-dns

helm install my-release \
  --set provider=pdns \
  --set pdns.apiUrl=API-URL \
  --set pdns.apiPort=API-Port \
  --set pdns.apiKey=API-Schlüssel \
  --set pdns.secretName=Name des Schlüssels
  
Die genauen Werte für die vier gesetzten Parameter sind dabei der Konfiguration der PowerDNS-Instanz zu entnehmen, die als Gegenspieler für "external-dns" fungiert. Im Zweifel lohnt es sich, den Admin des lokalen DNS-Servers ins Boot zu holen, um ordnungsgemäße DNS-Funktionalität zu gewährleisten. Im Anschluss geht es weiter mit oCIS selbst.

oCIS ausrollen

Eine große Stärke des Deployment-Ansatzes von Helm besteht darin, dass sich die Lösungen mehrerer Anbieter nahtlos kombinieren lassen, solange sie der Helm-Syntax folgen. So ist es aus oCIS-Sicht möglich, im eigenen Helm-Chart auf Funktionen zu setzen, die ExternalDNS, Cert-Manager und Nginx in Kubernetes überhaupt erst im Cluster etablieren. Und von eben dieser Option machen die Entwickler des Helm-Charts für oCIS ausgiebig Gebrauch. In den oCIS-Charts für Helm ist also die Möglichkeit, den Dienst mit Cert-Manager, Nginx und externem DNS-Dienst zu kombinieren, bereits angelegt. Entsprechend einfach gestaltet sich das Ausrollen von oCIS selbst. Dazu lädt der Administrator zunächst das gesamte OCIS-Git-Verzeichnis herunter:

https://github.com/owncloud/ocis-charts

Danach steht das Hinzufügen der Helm-Charts in die lokale Helm-Installation auf dem Plan. Der Befehl ist aus dem gerade durch "git" angelegten Ordner "ocis-charts" auszuführen, denn er bezieht sich auf die Datei "charts/ocis" darin:

helm install ocis ./charts/ocis

Ebenfalls innerhalb des "ocis-charts"-Ordners ist im nächsten Schritt eine Datei namens "values.yaml" anzulegen, die einige der Standardwerte der oCIS-Konfiguration im Helm-Chart überschreibt. Dabei geht es vor allem um den Hostnamen, den oCIS im Anschluss nutzen soll. Kasten 1 enthält ein vollständiges Beispiel für eine lauffähige Konfiguration.

Kasten 1: values.yaml-Beispiel

externalDomain: ocis.owncloud.test

ingress:
  enabled: true
  ingressClassName: nginx
  annotations:
    cert-manager.io/issuer: 'ocis-certificate-issuer'
  tls:
    - hosts:
        - ocis.example.net
      secretName: ocis-tls-certificate

extraResources:
  - |
    apiVersion: cert-manager.io/v1
    kind: Issuer
    metadata:
      name: ocis-certificate-issuer
    spec:
      acme:
        server: https://acme-v02.api.letsencrypt.org/directory
        email: test@example.net
        privateKeySecretRef:
          name: ocis-certificate-issuer
        solvers:
        - http01:
            ingress:
              class: nginx
			  
Kasten Ende

Im letzten Schritt muss "helm" noch einmal ran und die Änderungen der frisch angelegten "values.yaml" in das produktive Setup übernehmen. Das erfolgt wiederum aus dem Ordner "ocis-charts" heraus, den der Administrator per "git" zuvor angelegt hat:

helm upgrade --install --reset-values \
    ocis ./charts/ocis --values values.yaml
	
Im Anschluss zeigt "kubectl get pods --all-namespaces" die frisch entstehenden Container an, in denen alle zu oCIS gehörenden Komponenten laufen. Das Kommando "kubectl get services" sollte zudem einen Dienst auf der Kubernetes-Ebene des Typs "Nginx-Ingress" anzeigen, der als Schnittstelle zur Außenwelt fungiert. Dort gibt es das Feld "EXTERNAL-IP", das die IP-Adresse anzeigt, die Kubernetes der oCIS-Installation zugewiesen hat. Eine kurze Überprüfung der PowerDNS-Einstellungen im Beispiel sollte zusätzlich zeigen, dass Kubernetes im DNS-Server automatisch eine neuen Eintrag des Typs A für den neuen Dienst angelegt hat. Ist das der Fall, lässt sich die oCIS-Verbindung herstellen, indem man in das Adressfeld des Browsers "https://ocis.example.net" eingibt. So überprüft man implizit gleich auch, ob das Erstellen des SSL-Zertifikats funktioniert hat. Denn falls der Browser nun eine gültige SSL-Verschlüsselung anstelle einer Zertifikatswarnung oder gar einer SSL-Verschlüsselung anzeigt, hat "cert-manager" seinen Job ebenfalls erledigt. oCIS ist im Anschluss startklar.

Erweiterungen

Zusätzlich zum Standardeployment haben die Entwickler der Helm-Charts für oCIS viele Sonderfälle in ihrem Werk berücksichtigt. So bieten die Charts beispielsweise die Möglichkeit, eine externe Quelle für Benutzerdaten oder eine andere Speicherart einzusetzen. Genauere Informationen dazu enthält die Dokumentation der Helm-Charts unter [2]. Zusätzlich besteht die Möglichkeit, eine Instanz der Zeitreihendatenbank Prometheus (Abbildung 4) auszurollen, die oCIS im Anschluss umfassend überwacht. Hierzu genügt es, den Eintrag "extraResources" der "values.yaml" in Kasten 1 um einen zusätzlichen Eintrag zu erweitern:

extraResources:
  - |
    apiVersion: monitoring.coreos.com/v1
    kind: ServiceMonitor
    metadata:
      name: ocis-metrics
    spec:
      selector:
        matchLabels:
          ocis-metrics: enabled
      endpoints:
        - port: metrics-debug
          interval: 60s
          scrapeTimeout: 30s
          path: /metrics
		  
Vor dem erneuten Aufruf von "helm upgrade" analog zum zuvor beschriebenen Weg ist es auch noch nötig, die Prometheus-Integration für Kubernetes ("Custom Resource Definitions", kurz CRDs) zu installieren. Dazu lädt der Administrator zunächst mittels

git clone https://github.com/prometheus-operator/prometheus-operator

die Quellen dafür herunter und ruft im Anschluss mittels "bash scripts/generate-bundle.sh" ein Skript auf, das eine Datei namens "bundle.yaml" erstellt. Diese lässt sich danach mittels "kubectl apply -f bundle.yaml" in den Cluster einspielen. Weitere Möglichkeiten zur Überwachung mit Prometheus enthält darüber Hinaus die Prometheus-Dokumentation selbst.

Fazit

oCIS lässt sich mit Kubernetes schnell und unkompliziert ausrollen. Je nach gewählter Umgebung gibt es allerdings spezielle Konfigurationsparameter, die sich in einem Artikel wie diesem nicht umfassend abhandeln lassen. Wer beispielsweise oCIS auf AWS, Azure oder GCP ausrollen möchte, wird vermutlich eher die dortigen DNS-Server verwenden und die Konfiguration von "external-dns" entsprechend anpassen wollen. Darüber hinaus wird in vielen Fällen auch ein eigenes Storage-Backend für oCIS zur Anwendung kommen -- obgleich die Standardlösung auf Basis von Ceph, die oCIS verwendet, gar nicht verkehrt ist und sogar skalierbaren Speicher ermöglicht. Wer eine Sonderlocke braucht, findet in den Helm-Charts der oCIS-Entwickler aber durchaus die technische Grundlage, um auch diese zu realisieren.

Infos
[1] Helm: https://github.com/helm/helm/releases
[2] Dokumentation der Helm-Charts: https://doc.owncloud.com/ocis/next/deployment/container/orchestration/orchestration.html

Bild 1: ocis-helm_1.png
Screenshot: ownCloud
BU 1: ownCloud Infinite Scale alias oCIS ist eine neue ownCloud-Generation und kommt anders als der Vorgänger in Go daher. Mit wenigen Handgriffen lässt oCIS sich per Helm-Chart in Kubernetes ausrollen.

Bild 2: ocis-helm_2.png
Grafik: Nginx
BU 2: Nginx-Ingress erweitert Kubernetes um die Fähigkeit, dynamische Reverse Proxies für eingehende Dienste einzurichten. 

Bild 3: ocis-helm_3.png
Grafik: cert-manager
BU 3: Der Cert-Manager lädt das für oCIS so dringend benötigte SSL-Zertifikat automatisch von Let's Encrypt herunter. Verschlüsselung für oCIS auf der Dienstebene ist schließlich Pflicht.

Bild 4: ocis-helm_4.png
BU 4: Skalierbare Dienste wie oCIS benötigen neben Monitoring und Alerting auch Trending, um früh über Ressourcenengpässe informiert zu sein. oCIS leistet das in Kubernetes mittels Prometheus.
